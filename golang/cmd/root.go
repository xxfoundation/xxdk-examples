/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"crypto/ed25519"
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/nike/ecdh"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	mainNetUrl           = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/mainnet.json"
	partnerPublicKeyFlag = "partnerKey"
	partnerTokenFlag     = "partnerToken"
	timeoutFlag          = "timeout"
	ndfFlag              = "ndf"
	certFlag             = "cert"
	stateFlag            = "state"
	passwordFlag         = "password"
	messageFlag          = "message"
	countFlag            = "count"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "",
	Short: "Go Direct Messaging Example",
	Long: `A Direct Messaing example written in go, you can see other
examples at:

https://git.xx.network/xx_network/xxdk-examples`,

	Run: func(cmd *cobra.Command, args []string) {

		// Get an NDF
		ndfPath := viper.GetString(ndfFlag)
		certPath := viper.GetString(certFlag)
		ndf, err := os.ReadFile(ndfPath)
		if err != nil {
			certFile, _ := os.ReadFile(certPath)
			ndf, err = xxdk.DownloadAndVerifySignedNdfWithUrl(
				mainNetUrl, string(certFile))
			if err != nil {
				panic(fmt.Sprintf("+v", err))
			}
			os.WriteFile(ndfPath, ndf, os.FileMode(0777))
		}

		// Initialize or Load CMix net
		stateDir := viper.GetString(stateFlag)
		secret := []byte(viper.GetString(passwordFlag))
		stat, err := os.Stat(stateDir)
		if os.IsNotExist(err) || !stat.IsDir() {
			err = xxdk.NewCmix(string(ndf), stateDir, secret, "")
			if err != nil {
				panic(fmt.Sprintf("%+v", err))
			}
		}
		params := xxdk.GetDefaultCMixParams()
		net, err := xxdk.LoadCmix(stateDir, secret, params)
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}

		// Create or Load Direct Messaging Identity
		// NOTE: DM ID's are not storage backed, so we do the
		// storage here.
		ekv := net.GetStorage().GetKV()
		dmIDObj, err := ekv.Get("dmID", 0)
		if err != nil && ekv.Exists(err) {
			panic(fmt.Sprintf("%+v", err))
		}
		var dmID codename.PrivateIdentity
		if ekv.Exists(err) {
			dmID, err = codename.UnmarshalPrivateIdentity(
				dmIDObj.Data)
		} else {
			rng := net.GetRng().GetStream()
			defer rng.Close()
			dmID, err = codename.GenerateIdentity(rng)
			ekv.Set("dmID", &versioned.Object{
				Version:   0,
				Timestamp: time.Now(),
				Data:      dmID.Marshal(),
			})
		}
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
		dmToken := dmID.GetDMToken()
		pubKeyBytes := dmID.PubKey[:]

		fmt.Printf("DMPUBKEY: %s\n",
			base64.RawStdEncoding.EncodeToString(pubKeyBytes))
		fmt.Printf("DMTOKEN: %d\n", dmToken)

		partnerPubKey, partnerDMToken, ok := getDMPartner()
		if !ok {
			fmt.Printf("Setting dm destination to self\n")
			partnerPubKey = dmID.PubKey
			partnerDMToken = dmToken
		}

		fmt.Printf("DMRECVPUBKEY: %s\n",
			base64.RawStdEncoding.EncodeToString(partnerPubKey))
		fmt.Printf("DMRECVTOKEN: %d\n", partnerDMToken)

		recvCh := make(chan message.ID, 10)
		myReceiver := &receiver{
			recv:    recvCh,
			msgData: make(map[message.ID]*msgInfo),
			uuid:    0,
		}

		// nickname manager, sendTracker, and notifications manager
		// can be safely ignored unless you are doing something complex
		// in the UI. Expect these to become unnecessary for init
		// in favor of overloadable defaults.
		// Print user's reception ID
		storage := net.GetStorage()
		identity := storage.GetReceptionID()
		fmt.Printf("User ReceptionID: %s\n", identity)
		myNickMgr := dm.NewNicknameManager(identity, ekv)
		sendTracker := dm.NewSendTracker(ekv)
		transmissionID := net.GetTransmissionIdentity()
		sig := storage.GetTransmissionRegistrationValidationSignature()
		nm := notifications.NewOrLoadManager(transmissionID, sig,
			net.GetStorage().GetKV(), &notifications.MockComms{},
			net.GetRng())

		dmClient, err := dm.NewDMClient(&dmID, myReceiver, sendTracker,
			myNickMgr, nm, net.GetCmix(), ekv, net.GetRng(), nil)
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}

		err = net.StartNetworkFollower(5 * time.Second)
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		net.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		waitUntilConnected(connected)
		waitForRegistration(net, 0.85)

		// Message Sending
		go func() {
			for {
				text := viper.GetString(messageFlag)
				msgID, rnd, ephID, err := dmClient.SendText(
					partnerPubKey,
					partnerDMToken,
					text,
					cmix.GetDefaultCMIXParams())
				if err != nil {
					fmt.Printf("%+v\n", err)
				}
				fmt.Printf("DM Send: %v, %d, %v, %s\n", msgID,
					rnd.ID, ephID, text)
				time.Sleep(5 * time.Second)
			}
		}()

		// Message Reception Loop
		waitTime := viper.GetDuration(timeoutFlag) * time.Second
		maxReceiveCnt := viper.GetInt(countFlag)
		receiveCnt := 0
		for done := false; !done; {
			if maxReceiveCnt != 0 && receiveCnt >= maxReceiveCnt {
				done = true
				continue
			}
			select {
			case <-time.After(waitTime):
				done = true
			case m := <-recvCh:
				msg := myReceiver.msgData[m]
				selfStr := "Partner"
				if hmac.Equal(msg.senderKey[:],
					dmID.PubKey[:]) {
					selfStr = "Self"
				}
				fmt.Printf("Message received (%s, %s): %s\n",
					selfStr, msg.mType, msg.content)
				fmt.Printf("Message received: %s\n", msg)
				fmt.Printf("RECVDMPUBKEY: %s\n",
					base64.RawStdEncoding.EncodeToString(
						msg.partnerKey[:]))
				fmt.Printf("RECVDMTOKEN: %d\n", msg.dmToken)
				receiveCnt++
			}
		}
		if maxReceiveCnt == 0 {
			maxReceiveCnt = receiveCnt
		}
		fmt.Printf("Received %d/%d messages\n", receiveCnt,
			maxReceiveCnt)

		err = net.StopNetworkFollower()
		if err != nil {
			fmt.Printf(
				"Failed to cleanly close threads: %+v\n",
				err)
		}
		fmt.Printf("Client exiting!\n")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.Flags().StringP(partnerPublicKeyFlag, "d", "",
		"The public key of the dm partner (base64)")
	viper.BindPFlag(partnerPublicKeyFlag, rootCmd.Flags().Lookup(
		partnerPublicKeyFlag))

	rootCmd.Flags().Uint32P(partnerTokenFlag, "t", 0,
		"The token of the dm partner (integer)")
	viper.BindPFlag(partnerTokenFlag, rootCmd.Flags().Lookup(
		partnerTokenFlag))

	rootCmd.Flags().Int32P(timeoutFlag, "w", 30,
		"timeout time (default 5 seconds)")
	viper.BindPFlag(timeoutFlag, rootCmd.Flags().Lookup(
		timeoutFlag))

	rootCmd.Flags().StringP(ndfFlag, "n", "ndf.json",
		"Path to the ndf file")
	viper.BindPFlag(ndfFlag, rootCmd.Flags().Lookup(
		ndfFlag))

	rootCmd.Flags().StringP(certFlag, "c", "mainnet.crt",
		"Path to the mainnet certificate ndf signing file")
	viper.BindPFlag(certFlag, rootCmd.Flags().Lookup(
		certFlag))

	rootCmd.Flags().StringP(stateFlag, "s", "xxstate",
		"The xx network state directory")
	viper.BindPFlag(stateFlag, rootCmd.Flags().Lookup(
		stateFlag))

	rootCmd.Flags().StringP(messageFlag, "m", "Test Message",
		"The message to send")
	viper.BindPFlag(messageFlag, rootCmd.Flags().Lookup(
		messageFlag))

	rootCmd.Flags().StringP(passwordFlag, "p", "Hello",
		"The password to encrypt the state file")
	viper.BindPFlag(passwordFlag, rootCmd.Flags().Lookup(
		passwordFlag))

	rootCmd.Flags().Uint32P(countFlag, "", 0,
		"number of messages to wait for (default 0, infinite)")
	viper.BindPFlag(countFlag, rootCmd.Flags().Lookup(
		countFlag))
}

// getDMPartner parses the dmToken and dmPubKey flags from the command line
func getDMPartner() (ed25519.PublicKey, uint32, bool) {
	pubBytesStr := viper.GetString(partnerPublicKeyFlag)
	pubBytes, err := base64.RawStdEncoding.DecodeString(pubBytesStr)
	if err != nil {
		fmt.Printf("unable to read partner public key: %+v\n",
			err)
		return nil, 0, false
	}
	pubKey, err := ecdh.ECDHNIKE.UnmarshalBinaryPublicKey(pubBytes)
	if err != nil {
		fmt.Printf("unable to decode partner public key: %+v\n",
			err)
		return nil, 0, false
	}
	token := uint32(viper.GetUint32(partnerTokenFlag))
	return ecdh.EcdhNike2EdwardsPublicKey(pubKey), token, true
}

// waitUntilConnected waits until the network connects and also
// spins off a thread to monitor for network health
func waitUntilConnected(connected chan bool) {
	waitTimeout := time.Duration(viper.GetUint(timeoutFlag))
	timeoutTimer := time.NewTimer(waitTimeout * time.Second)
	isConnected := false
	// Wait until we connect or panic if we can't by a timeout
	for !isConnected {
		select {
		case isConnected = <-connected:
			fmt.Printf("Network Status: %v\n",
				isConnected)
		case <-timeoutTimer.C:
			panic(fmt.Errorf("timeout on connection after %s",
				waitTimeout*time.Second))
		}
	}

	// Now start a thread to empty this channel and update us
	// on connection changes for debugging purposes.
	go func() {
		prev := true
		for {
			select {
			case isConnected = <-connected:
				if isConnected != prev {
					prev = isConnected
					fmt.Printf(
						"Network Status Changed: %v\n",
						isConnected)
				}
				break
			}
		}
	}()
}

// waitForRegistration minimizes chance of msg send failures by
// ensuring we have registered our client with enough nodes.
func waitForRegistration(user *xxdk.Cmix, threshhold float32) {
	// After connection, make sure we have registered with
	// at least 85% of the nodes
	var err error
	for numReg, total := 0, 100; numReg < int(threshhold*float32(total)); {
		fmt.Printf("%d < %d\n", numReg,
			int(threshhold*float32(total)))
		time.Sleep(1 * time.Second)
		numReg, total, err = user.GetNodeRegistrationStatus()
		if err != nil {
			panic(fmt.Errorf("%+v", err))
		}

		fmt.Printf("Registering with nodes (%d/%d)...\n",
			numReg, total)
	}

}

// receiver implements DMReceiver's callback functions
// It uses a channel to signal messages have been
// inserted into a map
// This should generally be implemented with a database that is
// thread safe, which we enforce using a mutex lock in this implementation.
type receiver struct {
	recv    chan message.ID
	msgData map[message.ID]*msgInfo
	uuid    uint64
	sync.Mutex
}

type msgInfo struct {
	messageID  message.ID
	replyID    message.ID
	nickname   string
	content    string
	partnerKey ed25519.PublicKey
	senderKey  ed25519.PublicKey
	dmToken    uint32
	codeset    uint8
	timestamp  time.Time
	round      rounds.Round
	mType      dm.MessageType
	status     dm.Status
	uuid       uint64
}

func (mi *msgInfo) String() string {
	return fmt.Sprintf("[%v-%v] %s: %s", mi.messageID, mi.replyID,
		mi.nickname, mi.content)
}

func (r *receiver) receive(messageID message.ID, replyID message.ID,
	nickname, text string, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	r.Lock()
	defer r.Unlock()
	msg, ok := r.msgData[messageID]
	if !ok {
		r.uuid += 1
		msg = &msgInfo{
			messageID:  messageID,
			replyID:    replyID,
			nickname:   nickname,
			content:    text,
			partnerKey: partnerKey,
			senderKey:  senderKey,
			dmToken:    dmToken,
			codeset:    codeset,
			timestamp:  timestamp,
			round:      round,
			mType:      mType,
			status:     status,
			uuid:       r.uuid,
		}
		r.msgData[messageID] = msg
	} else {
		msg.status = status
	}
	go func() { r.recv <- messageID }()
	return msg.uuid
}

func (r *receiver) Receive(messageID message.ID,
	nickname string, text []byte, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	fmt.Printf("Receive: %v\n", messageID)
	return r.receive(messageID, message.ID{}, nickname, string(text),
		partnerKey, senderKey, dmToken, codeset, timestamp, round, mType, status)
}

func (r *receiver) ReceiveText(messageID message.ID,
	nickname, text string, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, status dm.Status) uint64 {
	fmt.Printf("ReceiveText: %v\n", messageID)
	return r.receive(messageID, message.ID{}, nickname, text,
		partnerKey, senderKey, dmToken, codeset, timestamp, round,
		dm.TextType, status)
}
func (r *receiver) ReceiveReply(messageID message.ID,
	reactionTo message.ID, nickname, text string,
	partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status dm.Status) uint64 {
	fmt.Printf("ReceiveReply: %v\n", messageID)
	return r.receive(messageID, reactionTo, nickname, text,
		partnerKey, senderKey, dmToken, codeset, timestamp, round,
		dm.TextType, status)
}
func (r *receiver) ReceiveReaction(messageID message.ID,
	reactionTo message.ID, nickname, reaction string,
	partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status dm.Status) uint64 {
	fmt.Printf("ReceiveReaction: %v\n", messageID)
	return r.receive(messageID, reactionTo, nickname, reaction,
		partnerKey, senderKey, dmToken, codeset, timestamp, round,
		dm.ReactionType,
		status)
}
func (r *receiver) UpdateSentStatus(uuid uint64, messageID message.ID,
	timestamp time.Time, round rounds.Round, status dm.Status) {
	r.Lock()
	defer r.Unlock()
	fmt.Printf("UpdateSentStatus: %v\n", messageID)
	msg, ok := r.msgData[messageID]
	if !ok {
		fmt.Printf("UpdateSentStatus msgID not found: %v\n",
			messageID)
		return
	}
	msg.status = status
}

func (r *receiver) DeleteMessage(message.ID, ed25519.PublicKey) bool {
	return true
}

func (r *receiver) GetConversation(ed25519.PublicKey) *dm.ModelConversation {
	return nil
}

func (r *receiver) GetConversations() []dm.ModelConversation {
	return nil
}
