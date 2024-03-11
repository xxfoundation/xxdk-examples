//
//  iOSExampleApp.swift
//  iOSExample
//
//  Created by Richard Carback on 3/4/24.
//

import SwiftUI
import SwiftData

@main
struct iOS_ExampleApp: App {
    @StateObject var logOutput = LogViewer()
    @StateObject var xxdk = XXDK(url: MAINNET_URL, cert: MAINNET_CERT)
    
    var sharedModelContainer: ModelContainer = {
        let schema = Schema([
            Item.self,
        ])
        let modelConfiguration = ModelConfiguration(schema: schema, isStoredInMemoryOnly: false)

        do {
            return try ModelContainer(for: schema, configurations: [modelConfiguration])
        } catch {
            fatalError("Could not create ModelContainer: \(error)")
        }
    }()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(logOutput)
                .environmentObject(xxdk)
        }
        .modelContainer(sharedModelContainer)
    }
}
