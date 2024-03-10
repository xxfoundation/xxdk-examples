'use client';

import { createContext, useState, useEffect, useContext, MouseEventHandler, useRef } from 'react';

import { Console, Hook, Unhook } from 'console-feed'

import XXNDF from './ndf.json'

import { CMix, DMClient, DMReceivedCallback, XXDKUtils } from '@/public/xxdk-wasm/dist/src';
import { Button } from '@nextui-org/button';
import { Input } from '@nextui-org/input';
import Dexie from 'dexie';
import { DBConversation, DBDirectMessage } from '@/public/xxdk-wasm/dist/src/types/db';
const xxdk = require('xxdk-wasm');

// XXContext is used to pass in "XXDKUtils", which
// provides access to all xx network functions to the children
 
const XXContext = createContext<XXDKUtils | null>(null);
const XXNet = createContext<CMix | null>(null);
export function XXNetwork({ children }: { children: React.ReactNode }) {
    const [XXDKUtils, setXXDKUtils] = useState<XXDKUtils | null>(null)
    const [XXCMix, setXXCMix] = useState<CMix | null>(null);

    useEffect(() => {
        // By default the library uses an s3 bucket endpoint to download at
        // https://elixxir-bins.s3-us-west-1.amazonaws.com/wasm/xxdk-wasm-[semver]
        // the wasm resources, but you can host them locally by
        // symlinking your public directory:
        //   cd public && ln -s ../node_modules/xxdk-wasm xxdk-wasm && cd ..
        // Then override with this function here:
        xxdk.setXXDKBasePath(window!.location.href + 'xxdk-wasm');
        xxdk.InitXXDK().then(async(xx: XXDKUtils) => {
            setXXDKUtils(xx)

            // Now set up cMix, while other examples download
            // you must hard code the ndf file for now in your application.          
            const ndf = JSON.stringify(XXNDF)

            // The statePath is a localStorage path that holds cMix xx network state
            const statePath = "xx"

            // Instantiate a user with the state directory password "Hello"
            const secret = Buffer.from("Hello");
            const cMixParamsJSON = Buffer.from("");

            console.log(secret)

            const stateExists = localStorage.getItem('cMixInitialized');
            if (stateExists === null || !stateExists) {
                await xx.NewCmix(ndf, statePath, secret, "")
                localStorage.setItem('cMixInitialized', 'true');
            }
            xx.LoadCmix(statePath, secret, cMixParamsJSON).then((net: CMix) => {
                setXXCMix(net)
            })
        });
    }, [])

    return (
        <XXContext.Provider value={XXDKUtils}>
            <XXNet.Provider value={XXCMix}>
            { children }
            </XXNet.Provider>
        </XXContext.Provider>
    )
}


export function XXLogs() {
    const [logs, setLogs] = useState([])

    useEffect(() => {
      const hookedConsole = Hook(
        window.console,
        (log) => setLogs((currLogs) => [...currLogs, log]),
        false
      )
      return () => Unhook(hookedConsole)
    }, [])
  
    return (
        <div className="flex [overflow-anchor:none]">
            <Console logs={logs} variant="dark" />
        </div>
    )
}


// XXDirectMessages is used to pass "XXDMReceiver", which
// stores callbacks of events from the xxdk api for
// direct messages
const XXDMReceiver = createContext<String[]>([]);
const XXDMClient = createContext<DMClient | null>(null);

export function XXDirectMessages({ children }: { children: React.ReactNode }) {
    const xx = useContext(XXContext)
    const xxNet = useContext(XXNet)

    const [dmReceiver, setDMReceiver] = useState<String[]>([]);
    const [dmClient, setDMClient] = useState<DMClient | null>(null);
    // NOTE: a ref is used instead of state because changes should not
    // cause a rerender, and also our handler function will need
    // to be able to access the db object when it is set.
    const dmDB = useRef<Dexie | null>(null);

    useEffect(() => {
        if (xx === null || xxNet === null) {
            return;
        }
        
        var dmIDStr = localStorage.getItem("MyDMID");
        if (dmIDStr === null) {
            console.log("Generating DM Identity...");
            // NOTE: This will be deprecated in favor of generateCodenameIdentity(...)
            dmIDStr = Buffer.from(xx.GenerateChannelIdentity(xxNet.GetID())).toString('base64');
            localStorage.setItem("MyDMID", dmIDStr);
        }
        console.log("Exported Codename Blob: " + dmIDStr);
        // Note: we parse to convert to Byte Array
        const dmID = new Uint8Array(Buffer.from(dmIDStr, 'base64'));

        // Web does not support notifications, so we use a dummy call
        var notifications = xx.LoadNotificationsDummy(xxNet.GetID());

        // DatabaseCipher encrypts using the given password, the max
        // size here the max for xx network DMs. 
        const cipher = xx.NewDatabaseCipher(xxNet.GetID(),
            Buffer.from("MessageStoragePassword"), 725);

        // The following handles events, namely to decrypt messages
        const onDmEvent = (eventType: number, data: Uint8Array) => {
            const msg = Buffer.from(data)
            console.log("onDmEvent called -> EventType: " + eventType + ", data: " + msg);

            dmReceiver.push(msg.toString('utf-8'));
            setDMReceiver([...dmReceiver]);

            const db = dmDB.current
            if (db !== null) {
                console.log("XXDB Lookup!!!!")
                // If we have a valid db object, we can
                // look up messages in the db and decrypt their contents
                const e = JSON.parse(msg.toString("utf-8"));
                Promise.all([
                    db.table<DBDirectMessage>("messages")
                        .where('id')
                        .equals(e.uuid)
                        .first(),
                    db.table<DBConversation>("conversations")
                        .filter((c) => c.pub_key === e.pubKey)
                        .last()
                ]).then(([message, conversation]) => {
                    if (!conversation) {
                        console.log(e);
                        console.error("XXDB Couldn't find conversation in database: " + e.pubKey);
                        return;
                    }
                    if (!message) {
                        console.log(e);
                        console.error("XXDB Couldn't find message in database: " + e.uuid);
                        return;
                    }

                    // You can tell if a message ID is new by it's id 
                    // (and you should be ordering them by date)
                    // For now we can just decrypt and print repeats
                    const plaintext = Buffer.from(cipher.Decrypt(message.text));
                    dmReceiver.push("Decrypted Message: " + plaintext.toString('utf-8'));
                    setDMReceiver([...dmReceiver]);
        
                });
            }
        }

        // Start a wasm worker for indexedDB that handles 
        // DM reads and writes and create DM object with it
        xxdk.dmIndexedDbWorkerPath().then((workerPath: string) => {
            // NOTE: important to explicitly convert to string here
            // will be fixed in future releases.
            const workerStr = workerPath.toString()
            console.log("DM Worker Path: " + workerPath.toString());
            xx.NewDMClientWithIndexedDb(xxNet.GetID(), notifications.GetID(),
                cipher.GetID(), workerStr, dmID,
                { EventUpdate: onDmEvent }).then((client) => {
                    console.log("DMTOKEN: " + client.GetToken());
                    console.log("DMPUBKEY: " + Buffer.from(client.GetPublicKey()).toString('base64'));

                    // Once we know our public key, that is the name of our database
                    // We have to remove the padding from base64 to get the db name
                    const dbName = Buffer.from(client.GetPublicKey()).toString('base64').replace(/={1,2}$/, '');
                    const db = new Dexie(dbName + "_speakeasy_dm")
                    db.open().then(() => {
                        console.log(db);
                        dmDB.current = db;
                    });

                    // Once all of our clients are loaded we can start
                    // listening to the network
                    xxNet.StartNetworkFollower(10000);
                    xxNet.WaitForNetwork(30000)

                    // When the network goes healthy, signal that to anything
                    // waiting on the client that it is ready.
                    setDMClient(client);
                });
        });
    }, [xx, xxNet]);

    return (
        <XXDMClient.Provider value={dmClient}>
            <XXDMReceiver.Provider value={dmReceiver}>
            { children }
            </XXDMReceiver.Provider>
        </XXDMClient.Provider>
    );
}

// XXDMSend
export async function XXDMSend(dm: DMClient, msg: string): Promise<boolean> {
    const myToken = dm.GetToken();
    const myPubkey = dm.GetPublicKey();

    return await dm.SendText(myPubkey, myToken, msg, 0, Buffer.from("")).then((sendReport) => {
        console.log(sendReport);
        return true;
    }).catch((err) => {
        console.log("could not send: " + err)
        return false
    });
}

export function XXMsgSender() {
    const dm = useContext(XXDMClient);
    const [msgToSend, setMessage] = useState<string>("");

    const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const newMsg = event.target.value;
        console.log(newMsg);
        setMessage(newMsg);
    }

    const handleSubmit = async () => {
        if (dm === null) {
            return;
        }
        if (await XXDMSend(dm, msgToSend)) {
            setMessage("");
        }
    }

    return (
        <>
        <div className="flex flex-grow p-2">
            <Input type="text" placeholder="Type message to send..."
                    value={msgToSend}
                    onChange={handleInputChange}/>
        </div>
        <div className="flex p-2">
            <Button size="md" color="primary" onClick={handleSubmit}>
            Submit
            </Button>
        </div>
        </>
    )
}

// XXDirectMessagesReceived is just a buffer of received event messages
export function XXDirectMessagesReceived() {
    const msgs = useContext(XXDMReceiver);

    if (msgs === null || msgs.length == 0) {
        return (
            <div>Nothing yet...</div>
        )
    }

    const msgOut = msgs.map(m => <div className="[overflow-anchor:none]">{m}</div>);
    return (
        msgOut
    )
}