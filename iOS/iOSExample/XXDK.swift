//
//  XXDK.swift
//  iOSExample
//
//  Created by Richard Carback on 3/6/24.
//

import Foundation

import Bindings
import Kronos

// NDF is the configuration file used to connect to the xx network. It
// is a list of known hosts and nodes on the network.
// A new list is downloaded on the first connection to the network
public var MAINNET_URL = "https://elixxir-bins.s3.us-west-1.amazonaws.com/ndf/mainnet.json"
// This resolves to "Resources/mainnet.crt" in the project folder for iOSExample
public var MAINNET_CERT = Bundle.main.path(forResource: "mainnet", ofType: "crt") ?? "unknown resource path"

@MainActor
public class XXDK: ObservableObject {
    private var networkUrl = MAINNET_URL
    private var networkCert = MAINNET_CERT
    private var stateDir: URL
    
    // These are initialized after loading
    @Published var ndf: Data?
    private var net: Bindings.BindingsCmix?
    @Published var DM: Bindings.BindingsDMClient?
    // This will not start receiving until the network follower starts
    @Published var dmReceiver = DMReceiver()
    
    init(url: String, cert: String) {
        networkUrl = url
        networkCert = cert

        let netTime = NetTime()
        // xxdk needs accurate time to connect to the live network
        Bindings.BindingsSetTimeSource(netTime)

        // Note: this will resolve to the documents folder on Mac OS
        // or the app's local data folder on iOS.
        do {
            let basePath = try FileManager.default.url(
                for: .documentDirectory,
                in: .userDomainMask,
                appropriateFor: nil,
                create: false)
            stateDir = basePath.appendingPathComponent("xxAppState")
            if !FileManager.default.fileExists(atPath: stateDir.path) {
                try FileManager.default.createDirectory(at: stateDir, withIntermediateDirectories: true)
            }
            stateDir = stateDir.appendingPathComponent("ekv")
        } catch let err {
            fatalError("failed to get state directory: " + err.localizedDescription)
        }

    }
    
    func load() async {
        ndf = downloadNDF(url: self.networkUrl, certFilePath: self.networkCert)
        
        // NOTE: Secret should be pulled from keychain
        let secret = "Hello".data
        // NOTE: Empty string forces defaults, these are settable but it is recommended that you use the defaults.
        let cmixParamsJSON = "".data
        if !FileManager.default.fileExists(atPath: stateDir.path) {
            var err: NSError?
            Bindings.BindingsNewCmix(ndf?.utf8, stateDir.path, secret, "", &err)
            if err != nil {
                fatalError("could not create new Cmix: " + err!.localizedDescription)
            }
        }
        var err: NSError?
        net = Bindings.BindingsLoadCmix(stateDir.path, secret, cmixParamsJSON, &err)
        if err != nil {
            fatalError("could not load Cmix: " + err!.localizedDescription)
        }
        
        let receptionID = net?.getReceptionID()!.base64EncodedString()
        print("cMix Reception ID: \(receptionID ?? "<nil value>")")
        
        let dmID: Data
        do {
            dmID = try net!.ekvGet("MyDMID")
        } catch {
            print("Generating DM Identity...")
            // NOTE: This will be deprecated in favor of generateCodenameIdentity(...)
            dmID = Bindings.BindingsGenerateChannelIdentity(net!.getID(), &err)!;
            if err != nil {
                fatalError("could not generate codename id: " + err!.localizedDescription)
            }
            print("Exported Codename Blob: " +
                  dmID.base64EncodedString())
            do {
                try net!.ekvSet("MyDMID", value: dmID)
            } catch let error {
                fatalError("could not set ekv: " + error.localizedDescription)
            }
        }
        print("Exported Codename Blob: " +
              dmID.base64EncodedString())
        
        let notifications = Bindings.BindingsLoadNotifications(net!.getID(), &err)
        if err != nil {
            fatalError("could not load notifications: " + err!.localizedDescription)
        }

        let receiverBuilder = DMReceiverBuilder(receiver: dmReceiver)

        //Note: you can use `newDmManagerMobile` here instead if you want to work with
        //an SQLite database.
        // This interacts with the network and requires an accurate clock to connect or you'll see
        // "Timestamp of request must be within last 5 seconds." in the logs.
        // If you have trouble shutdown and start your emulator.
        DM = Bindings.BindingsNewDMClient(net!.getID(), (notifications?.getID())!,
                                              dmID, receiverBuilder, dmReceiver, &err)
        if err != nil {
            fatalError("could not load dm client: " + err!.localizedDescription)
        }

        print("DMPUBKEY: \(DM?.getPublicKey()?.base64EncodedString() ?? "empty pubkey")")
        print("DMTOKEN: \(DM?.getToken() ?? 0)")

        
        do {
            try net!.startNetworkFollower(5000)
            net!.wait(forNetwork: 30000)
        } catch let error {
            fatalError("cannot start network: " + error.localizedDescription)
        }
    }
    
    func sendDM(msg: String) {
        // Note: These would get set by the view loading the conversation in a real app, for now we just
        // set things up to message ourselves.
        let dmPubKey = DM?.getPublicKey()
        let token = Int32((DM?.getToken())!)
        
        do {
            try DM!.sendText(dmPubKey, partnerToken: token, message: msg, leaseTimeMS: 0, cmixParamsJSON: "".data)
        } catch let error {
            fatalError("Unable to send: " + error.localizedDescription)
        }
    }
    
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
    func downloadNDF(url: String, certFilePath: String) -> Data {
        let certString: String
        do {
            certString = try String(contentsOfFile: certFilePath)
        } catch let error {
            fatalError("Missing network certificate, please include a mainnet, testnet," +
                       "or localnet certificate in the Resources folder: " + error.localizedDescription)
        }

        var err: NSError?
        let ndf = Bindings.BindingsDownloadAndVerifySignedNdfWithUrl(url, certString, &err)
        if err != nil {
            fatalError("DownloadAndverifySignedNdfWithUrl(\(url), \(certString)) error: " + err!.localizedDescription)
        }
        // Golang functions uss a `return val or nil, nil or err` pattern, so ndf will be valid data after
        // checking if the error has anything in it.
        return ndf!
    }

}

// These are common helpers extending the string class which are essential for working with XXDK
extension StringProtocol {
    var data: Data { .init(utf8) }
    var bytes: [UInt8] { .init(utf8) }
}
extension DataProtocol {
    var utf8: String { String(decoding: self, as: UTF8.self) }
}

class NetTime: NSObject, Bindings.BindingsTimeSourceProtocol {
    override init() {
        super.init()
        Kronos.Clock.sync()
    }
    
    func nowMs() -> Int64 {
        let curTime = Kronos.Clock.now
        if curTime == nil {
            Kronos.Clock.sync()
            return Int64(Date.now.timeIntervalSince1970)
        }
        return Int64(Kronos.Clock.now!.timeIntervalSince1970)
    }
}


// DmReceiverBuilder is a wrapper for a stateful (database-based)
// DMReceiver implementation.
class DMReceiverBuilder: NSObject, Bindings.BindingsDMReceiverBuilderProtocol {
    private var r: DMReceiver
    
    init(receiver: DMReceiver) {
        self.r = receiver
        super.init()
    }
    
    func build(_ path: String?) -> (any BindingsDMReceiverProtocol)? {
        return r
    }
}


struct ReceivedMessage: Identifiable {
    var Msg: String
    var id = UUID()
}

// DMReceiver's are callbacks for message processing. These include
// message reception and retrieval of specific data to process a message.
// DmCallbacks are events that signify the UI should be updated
// for full details see the docstrings or the "bindings" folder
// inside the core codebase.
// We implement them both inside the same object for convenience of passing updates to the UI.
// Your implementation may vary based on your needs.
class DMReceiver: NSObject, ObservableObject, Bindings.BindingsDMReceiverProtocol, Bindings.BindingsDmCallbacksProtocol {
    @Published var msgBuf: [ReceivedMessage] = []
    private var msgCnt: Int64 = 0
    
    func eventUpdate(_ eventType: Int64, jsonData: Data?) {
        msgBuf.append(ReceivedMessage(Msg: "Received Event id \(eventType)"))
    }
        
    func deleteMessage(_ messageID: Data?, senderPubKey: Data?) -> Bool {
        msgBuf.append(ReceivedMessage(Msg: "Delete message: " +
                      "\(messageID?.base64EncodedString() ?? "empty id"), " +
                      "\(senderPubKey?.base64EncodedString() ?? "empty pubkey")"))
        return true
    }
    
    func getConversation(_ senderPubKey: Data?) -> Data? {
        msgBuf.append(ReceivedMessage(Msg: "getConversation: \(senderPubKey?.base64EncodedString() ?? "empty pubkey")"))
        return "".data
    }
    
    func getConversations() -> Data? {
        msgBuf.append(ReceivedMessage(Msg: "getConversations"))
        return "[]".data
    }
    
    func receive(_ messageID: Data?, nickname: String?, text: Data?, partnerKey: Data?, senderKey: Data?, dmToken: Int32, codeset: Int, timestamp: Int64, roundId: Int64, mType: Int64, status: Int64) -> Int64 {
        msgBuf.append(ReceivedMessage(Msg: "\(senderKey?.base64EncodedString() ?? "empty pubkey"): \(text?.utf8 ?? "empty text")"))
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }
    
    func receiveReaction(_ messageID: Data?, reactionTo: Data?, nickname: String?, reaction: String?, partnerKey: Data?, senderKey: Data?, dmToken: Int32, codeset: Int, timestamp: Int64, roundId: Int64, status: Int64) -> Int64 {
        msgBuf.append(ReceivedMessage(Msg:  "\(senderKey?.base64EncodedString() ?? "empty pubkey"): \(reaction ?? "empty text")"))
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }
    
    func receiveReply(_ messageID: Data?, reactionTo: Data?, nickname: String?, text: String?, partnerKey: Data?, senderKey: Data?, dmToken: Int32, codeset: Int, timestamp: Int64, roundId: Int64, status: Int64) -> Int64 {
        msgBuf.append(ReceivedMessage(Msg: "\(senderKey?.base64EncodedString() ?? "empty pubkey"): \(text ?? "empty text")"))
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }
    
    func receiveText(_ messageID: Data?, nickname: String?, text: String?, partnerKey: Data?, senderKey: Data?, dmToken: Int32, codeset: Int, timestamp: Int64, roundId: Int64, status: Int64) -> Int64 {
        msgBuf.append(ReceivedMessage(Msg: "\(senderKey?.base64EncodedString() ?? "empty pubkey"): \(text ?? "empty text")"))
        // Note: this should be a UUID in your database so
        // you can uniquely identify the message.
        msgCnt += 1;
        return msgCnt;
    }
    
    func updateSentStatus(_ uuid: Int64, messageID: Data?, timestamp: Int64, roundID: Int64, status: Int64) {
        msgBuf.append(ReceivedMessage(Msg: "Message sent status update: \(uuid) -> \(status), \(roundID)"))
    }
    
    
}
