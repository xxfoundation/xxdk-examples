
import {Input, Button} from "@nextui-org/react";
import { XXNetwork, XXLogs, XXDirectMessages, XXDirectMessagesReceived, XXDMSend, XXMsgSender } from "./xxdk";

export default function Home() {
  return (
    <main className="flex flex-col min-h-screen flex-col items-center p-10">
      <XXNetwork>
      <XXDirectMessages>
        <div className="flex-grow flex-col max-h-96 overflow-y-auto overflow-x-wrap w-4/5 border border-gray-300 m-0 [overflow-anchor:none]">
          <p className="flex w-full justify-center">Received Messages</p>
          <XXDirectMessagesReceived />
          <div id="anchor2" className="h-1 [overflow-anchor:auto]"></div>
        </div>
        <div className="flex w-4/5 border border-gray-300 m-1">
          <XXMsgSender />
        </div>
      </XXDirectMessages>
      </XXNetwork>
    </main>
  );
}


