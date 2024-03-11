//
//  LogViewer.swift
//  iOS Example
//
//  Created by Richard Carback on 3/1/24.
//

import Foundation


struct LogMessage: Identifiable {
    var Msg: String
    var id = UUID()
}

// LogViewer observes messages to stdout and stderr and sends them to your callback
// function
@MainActor
public class LogViewer: ObservableObject {
    @Published var Messages: [LogMessage] = [LogMessage(Msg: "LogMessages")]
    private var size: Int
    private var count: UInt = 1
    
    init(numLines: Int = 1000) {
        size = numLines
        Messages.reserveCapacity(size)
        
        let stdoutFile = OpenFileCloner(fd: STDOUT_FILENO).Output
        // Redirect stderr to stdout. I didn't figure out why, but you can't
        // do a OpenFileCloner call on both the way you might expect but this
        // does work if you need it. In most cases your app is logging to stdout,
        // so you don't need to be worried here and can delete it.
        dup2(STDOUT_FILENO, STDERR_FILENO)

        stdoutFile.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            if data.isEmpty {
                return
            }
            DispatchQueue.main.async {
                // This is dumb but we're processing each line,
                // stripping off the OSLOG prefix up to the first tab (\t)
                // then adding it to the buffer
                let str = String(data: data, encoding: String.Encoding.utf8) ?? "empty data"
                let lines = str.split(separator: "\n")
                for line in lines {
                    let idx = line.firstIndex(of: "\t")
                    var start = line.startIndex
                    // extra special stupid of making sure we use the index
                    // after the one we found. I couldn't get +1 to work and
                    // I don't care enough to figure out if
                    // there's a better way, I imagine there is
                    if idx != nil {
                        start = line.index(after: idx!)
                    }
                    
                    // Now we add to the published messages object
                    // and reduce it's size if it gets too big.
                    self.Messages.append(LogMessage(Msg: String(line[start...])))
                    self.count += 1
                    while self.Messages.count > self.size {
                        self.Messages.remove(at: 0)
                    }
                }
            }
        }
    }
}

// OpenFileCloner takes an existing file descriptor, `fd`, and creates
// a pipe to allow you to read what is being written to it from a
// separate pipe. You would typically use this with STDOUT_FILENO
// or STDERR_FILENO to read all output an app is writing to it.
//
// As part of the cloning, a background process is set up to
// copy all writes to the original `fd`. The original `fd` can be used
// to write to it if needed.
@MainActor
class OpenFileCloner : ObservableObject {
    private var input: Pipe
    private var origOut: Pipe
    var Output: Pipe
    
    init(fd: Int32) {
        input = Pipe()
        origOut = Pipe()
        Output = Pipe()

        // ensure that std out and err are unbuffered, which
        // sometimes changes depending on platform/version
        if fd == STDOUT_FILENO {
            setvbuf(stdout, nil, _IONBF, 0)
        }
        if fd == STDERR_FILENO {
            setvbuf(stderr, nil, _IONBF, 0)
        }

        // First, dup2 the target fd to our output pipe. This means
        // that the existing fd is the origOutput pipe's
        dup2(fd, origOut.fileHandleForWriting.fileDescriptor)

        // Now replace the fd with the input pipe, new log
        // writes will come in through the input and we will
        // copy them into the output pipes.
        dup2(input.fileHandleForWriting.fileDescriptor, fd)
        
        // monitor the input so that anything written to it is
        // also copied to the output pipes so the user can read it.
        // listen in to the readHandle notification
        // NOTE: readabilityHandler seemed better than the observer pattern
        // with notifications BUT note that it will trigger even when there is no
        // data to read.
        input.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            if data.isEmpty {
                return
            }
            DispatchQueue.main.async { [self] in
                self.origOut.fileHandleForWriting.write(data)
                self.Output.fileHandleForWriting.write(data)
            }
        }
    }
}

