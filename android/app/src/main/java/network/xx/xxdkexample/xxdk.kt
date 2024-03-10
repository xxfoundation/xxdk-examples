package network.xx.xxdkexample

import android.content.Context
import android.util.Log
import android.widget.TextView
import bindings.Bindings
import bindings.DMReceiver
import bindings.DMReceiverBuilder
import com.lyft.kronos.AndroidClockFactory
import com.lyft.kronos.KronosClock
import java.util.Base64

// NDF is the configuration file used to connect to the xx network. It
// is a list of known hosts and nodes on the network.
// A new list is downloaded on the first connection to the network
const val NDF_URL_MAIN = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/mainnet.json"

// downloadNdf uses the mainnet URL to download and verify the
// network definition file for the xx network.
// As of this writing, using the xx network is free and using the public
// network is OK. Check the xx network docs for updates.
// You can test locally, with the integration or localenvironment
// repositories with their own ndf files here:
//  * https://git.xx.network/elixxir/integration
//  * https://git.xx.network/elixxir/localenvironment
// integration will run messaging tests against a local network,
// and localenvironment will run a fixed network local to your machine.
fun downloadNdf(context: Context, url: String = NDF_URL_MAIN): String {
    val r = context.resources
    val reader = r.openRawResource(R.raw.mainnet).bufferedReader()
    val certFile = reader.readText()
    val ndf = Bindings.downloadAndVerifySignedNdfWithUrl(url, certFile)
    return String(ndf)
}

// timeSource helps by using ntp to keep the clock in sync.
// notably on startup you would need accurate time until it has
// a chance to complete the first sync. You could modify this to be
// more aggressive but it would affect app load time.
class timeSource(appContext: Context) : bindings.TimeSource {
    var kronosClock: KronosClock

    init {
        kronosClock = AndroidClockFactory.createKronosClock(appContext)
        kronosClock.syncInBackground()
    }

    override fun nowMs(): Long {
        return kronosClock.getCurrentTimeMs()
    }

}

// DmCallbacks are events that signify the UI should be updated
// for full details see the docstrings or the "bindings" folder
// inside the core codebase.
class DmEvents(private val view: TextView) : bindings.DmCallbacks {
    override fun eventUpdate(eventType: Long, jsonData: ByteArray?) {
        view.append("Received Event id $eventType\n")
    }
}

// DmReceiverBuilder is a wrapper for a stateful (database-based)
// DMReceiver implementation.
class DmReceiverBuilder(private val receiver: bindings.DMReceiver) : bindings.DMReceiverBuilder {
    override fun build(path: String?): DMReceiver {
        return receiver
    }

}

// DMReceiver's are callbacks for message processing. These include
// message reception and retrieval of specific data to process a message.
class DmReceiver(private val view: TextView) : bindings.DMReceiver {
    private var msgCnt: Long = 0;
    override fun deleteMessage(messageID: ByteArray?, senderPubKey: ByteArray?): Boolean {
        view.append("Delete message: " +
                    "${Base64.getEncoder().encodeToString(messageID)}, " +
                    "${Base64.getEncoder().encodeToString(senderPubKey)}\n")
        return true
    }

    // getConversation allows retrieval of nickname and public key for a give senderPubKey
    // You should return the following json data structure in Go:
    // type ModelConversation struct {
	//      Pubkey         []byte `json:"pub_key"`
	//      Nickname       string `json:"nickname"`
	//      Token          uint32 `json:"token"`
	//      CodesetVersion uint8  `json:"codeset_version"`
    //
	//      // Deprecated: KV is the source of truth for blocked users.
	//      BlockedTimestamp *time.Time `json:"blocked_timestamp"`
    // }
    override fun getConversation(senderPubKey: ByteArray?): ByteArray {
        view.append("getConversation: ${Base64.getEncoder().encodeToString(senderPubKey)}")
        return "".toByteArray()
    }

    // getConversations returns all conversations stored on the UI side
    override fun getConversations(): ByteArray {
        view.append("getConversations")
        return "[]".toByteArray()
    }

    // Receive a raw message
    override fun receive(
        messageID: ByteArray?,
        nickname: String?,
        text: ByteArray?,
        partnerKey: ByteArray?,
        senderKey: ByteArray?,
        dmToken: Int,
        codeset: Long,
        timestamp: Long,
        roundId: Long,
        mType: Long,
        status: Long
    ): Long {
        view.append("${Base64.getEncoder().encodeToString(senderKey)}: $text\n")
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }

    override fun receiveReaction(
        messageID: ByteArray?,
        reactionTo: ByteArray?,
        nickname: String?,
        reaction: String?,
        partnerKey: ByteArray?,
        senderKey: ByteArray?,
        dmToken: Int,
        codeset: Long,
        timestamp: Long,
        roundId: Long,
        status: Long
    ): Long {
        view.append("${Base64.getEncoder().encodeToString(senderKey)}: $reaction\n")
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }

    override fun receiveReply(
        messageID: ByteArray?,
        reactionTo: ByteArray?,
        nickname: String?,
        text: String?,
        partnerKey: ByteArray?,
        senderKey: ByteArray?,
        dmToken: Int,
        codeset: Long,
        timestamp: Long,
        roundId: Long,
        status: Long
    ): Long {
        view.append("${Base64.getEncoder().encodeToString(senderKey)}: $text\n")
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }

    override fun receiveText(
        messageID: ByteArray?,
        nickname: String?,
        text: String?,
        partnerKey: ByteArray?,
        senderKey: ByteArray?,
        dmToken: Int,
        codeset: Long,
        timestamp: Long,
        roundId: Long,
        status: Long
    ): Long {
        view.append("${Base64.getEncoder().encodeToString(senderKey)}: $text\n")
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }

    override fun updateSentStatus(
        uuid: Long,
        messageID: ByteArray?,
        timestamp: Long,
        roundID: Long,
        status: Long
    ) {
        view.append("Message sent status update: $uuid -> $status, $roundID\n")
    }
}