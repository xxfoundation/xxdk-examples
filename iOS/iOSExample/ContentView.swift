//
//  ContentView.swift
//  iOS Example
//
//  Created by Richard Carback on 2/29/24.
//

import SwiftUI
import SwiftData

struct ContentView: View {
    @State private var SendMessageTextInput: String = ""
    @EnvironmentObject var xxdk: XXDK
    @EnvironmentObject var logOutput: LogViewer
    
    var body: some View {
        VStack {
            ViewThatFits {
                ScrollView {
                    ForEach(logOutput.Messages) { line in
                        Text(line.Msg)
                    }
                }.defaultScrollAnchor(.bottom)
            }
            .padding()
            .frame(maxWidth: .infinity)
            .border(.primary)
            ViewThatFits {
                ScrollView {
                    Text("Message Received Viewer")
                    ForEach(xxdk.dmReceiver.msgBuf) { msg in
                        Text(msg.Msg)
                    }
                }.defaultScrollAnchor(.bottom)
            }
            .padding()
            .frame(maxWidth: .infinity)
            .border(.primary)
            HStack(alignment: .bottom) {
                TextField ("Enter Message to Send",
                           text: $SendMessageTextInput)
                .onKeyPress(.return, action: {
                    xxdk.sendDM(msg: SendMessageTextInput)
                    _SendMessageTextInput.wrappedValue = ""
                    return KeyPress.Result.handled
                })
                .textFieldStyle(.roundedBorder)
                Button(action: {
                    xxdk.sendDM(msg: SendMessageTextInput)
                    _SendMessageTextInput.wrappedValue = ""
                }, label: {
                        Text("Send")
                })
                .buttonStyle(.borderedProminent)
            }.padding()
        }.padding()
        .onAppear(perform: {
            Task {
                await xxdk.load()
            }
        })
    }
}

#Preview {
    ContentView()
        .modelContainer(for: Item.self, inMemory: true)
}

