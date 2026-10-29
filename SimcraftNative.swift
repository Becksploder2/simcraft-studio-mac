import SwiftUI
import AppKit
import UniformTypeIdentifiers

@main struct SimcraftNative: App {
    var body: some Scene {
        WindowGroup("Simcraft Studio") { StudioView().frame(minWidth: 950, minHeight: 650) }
    }
}
struct StudioView: View {
    let tabs = ["Character Studio","Hair & Glasses","Clothing SmartAssist","CC Library","Face Texture Studio","Glasses Creator","CAS Mod Studio","AI Mod Studio","UV Face Alignment","Buildcraft","Settings"]
    @State private var selected = "AI Mod Studio"
    @State private var modName = ""
    @State private var category = "Gameplay interactions"
    @State private var details = ""
    @State private var compatibility = true
    @State private var photoURL: URL?
    @State private var templateGender = "F"
    @State private var templateNumber = 1
    @State private var glasses = "Yes"
    @State private var hair = "Match reference"
    @State private var occasion = "Everyday"
    @State private var styles: Set<String> = []
    @State private var catalogStatus = "No catalog imported"
    @State private var catalogPreview = ""
    @State private var casStatus = "CAS not checked"
    @State private var casCaptureURL: URL?
    @State private var backgroundURL: URL?
    @State private var updateStatus = "Update status: not checked"
    @State private var message = "Project specification only; not an installable mod."
    var body: some View {
        NavigationSplitView {
            List(selection: $selected) { ForEach(tabs, id: \.self) { item in Text(item).tag(item) } }
                .navigationTitle("Simcraft Studio")
                .frame(minWidth: 220)
        } detail: {
            ZStack {
                if let url = Bundle.main.url(forResource: "Default CAS Room", withExtension: "png", subdirectory: "Backgrounds"),
                   let room = NSImage(contentsOf: url) {
                    GeometryReader { geometry in
                        Image(nsImage: room).resizable().scaledToFill()
                            .frame(width: geometry.size.width, height: geometry.size.height)
                            .clipped()
                    }.ignoresSafeArea()
                    Color.black.opacity(0.62).ignoresSafeArea()
                }
            VStack(alignment: .leading, spacing: 18) {
                Text(selected).font(.largeTitle).bold()
                if selected == "AI Mod Studio" {
                    TextField("Mod name", text: $modName)
                    Picker("Mod category", selection: $category) {
                        ForEach(["Gameplay interactions","School and careers","CAS and accessories","Objects and tuning"], id: \.self) { Text($0) }
                    }
                    Text("Describe your mod")
                    TextEditor(text: $details).frame(height: 190).border(.gray)
                    Toggle("Include compatibility checks", isOn: $compatibility)
                    Button("Export mod project JSON") { exportProject() }.disabled(details.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    Text(message).foregroundStyle(.secondary)
                } else if selected == "Character Studio" {
                    Button("Import reference photo") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [.png, .jpeg, .bmp]
                        if picker.runModal() == .OK { photoURL = picker.url }
                    }
                    if let url = photoURL, let image = NSImage(contentsOf: url) {
                        Image(nsImage: image).resizable().scaledToFit().frame(maxHeight: 310)
                    }
                    Picker("Template group", selection: $templateGender) {
                        Text("Female").tag("F"); Text("Male").tag("M")
                    }
                    Picker("Template", selection: $templateNumber) {
                        ForEach(1...19, id: \.self) { n in Text("\(templateGender)\(n)").tag(n) }
                    }
                    HStack {
                        Button("Detect The Sims 4") { detectGame() }
                        Button("Capture Sims 4 window") { captureGame() }
                    }
                    Text(casStatus).foregroundStyle(.secondary)
                    Button("Choose CAS workspace background") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [.png, .jpeg, .bmp]
                        if picker.runModal() == .OK { backgroundURL = picker.url }
                    }
                    if let url = backgroundURL, let image = NSImage(contentsOf: url) {
                        Image(nsImage: image).resizable().scaledToFit().frame(maxHeight: 180)
                    }
                    Text("Background is a local preview; this does not replace the in-game CAS background.").foregroundStyle(.secondary)
                    Text("Window detection and capture only. Automatic CAS morphing is not yet implemented.").foregroundStyle(.secondary)
                } else if selected == "Hair & Glasses" {
                    Picker("Does your Sim wear glasses?", selection: $glasses) {
                        ForEach(["Yes","Sometimes","No"], id: \.self) { Text($0) }
                    }
                    Picker("Hair approach", selection: $hair) {
                        ForEach(["Match reference","Suggest flattering styles","Manual choice"], id: \.self) { Text($0) }
                    }
                    Text("Recommendations require an imported CAS catalog.").foregroundStyle(.secondary)
                } else if selected == "Clothing SmartAssist" {
                    Picker("Outfit occasion", selection: $occasion) {
                        ForEach(["Everyday","Formal","Athletic","Sleep","Party","Swimwear","Hot Weather","Cold Weather"], id: \.self) { Text($0) }
                    }
                    Text("Styles").font(.headline)
                    ScrollView {
                        ForEach(["Sporty","Basics","Outdoorsy","Polished","Preppy","Rocker","Country","Hipster","Boho","Streetwear","Gamer","Gala","Africana"], id: \.self) { style in
                            Toggle(style, isOn: Binding(get: { styles.contains(style) }, set: { enabled in if enabled { styles.insert(style) } else { styles.remove(style) } }))
                        }
                    }.frame(maxHeight: 280)
                    Button("Export outfit plan") { exportOutfit() }
                } else if selected == "CC Library" {
                    Button("Import JSON or CSV catalog") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [.json, .commaSeparatedText]
                        if picker.runModal() == .OK, let url = picker.url {
                            do {
                                let content = try String(contentsOf: url, encoding: .utf8)
                                if url.pathExtension.lowercased() == "json" {
                                    _ = try JSONSerialization.jsonObject(with: Data(content.utf8))
                                }
                                catalogPreview = String(content.prefix(16000))
                                catalogStatus = "Catalog loaded: \(url.lastPathComponent)"
                            } catch { catalogStatus = "Import failed: \(error.localizedDescription)" }
                        }
                    }
                    Text(catalogStatus)
                    ScrollView { Text(catalogPreview).font(.system(.caption, design: .monospaced)).textSelection(.enabled) }
                } else if selected == "Settings" {
                    Text("Native desktop preview. The HTML workspace is not used by this interface.")
                    Button("Check for updates") { Task { await checkForUpdates() } }
                    Text(updateStatus).foregroundStyle(.secondary)
                    Text("Photo import, manual template selection, preferences, outfit plans, CC catalog import and AI mod specifications are available.")
                } else {
                    Text("Native migration in progress. The previous HTML tool has not yet been ported.").foregroundStyle(.secondary)
                }
                Spacer()
            }.padding(28).frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }
    @MainActor func checkForUpdates() async {
        updateStatus = "Checking GitHub releases..."
        guard let url=URL(string:"https://api.github.com/repos/Becksploder2/simcraft-studio-mac/releases/latest") else { return }
        do {
            var request=URLRequest(url:url)
            request.setValue("SimcraftStudio/1.0.6", forHTTPHeaderField:"User-Agent")
            let (data,_) = try await URLSession.shared.data(for:request)
            guard let json=try JSONSerialization.jsonObject(with:data) as? [String:Any],
                  let tag=json["tag_name"] as? String else { updateStatus="Invalid release information"; return }
            let latest=tag.trimmingCharacters(in:CharacterSet(charactersIn:"vV"))
            if latest.compare("1.0.6",options:.numeric) != .orderedDescending {
                updateStatus="Up to date (v1.0.6)."; return
            }
            updateStatus="Update available: \(tag)"
            let alert=NSAlert()
            alert.messageText="Simcraft Studio \(tag) is available"
            alert.informativeText="Open the official Mac release page?"
            alert.addButton(withTitle:"Open release")
            alert.addButton(withTitle:"Later")
            if alert.runModal() == .alertFirstButtonReturn,
               let release=URL(string:"https://github.com/Becksploder2/simcraft-studio-mac/releases/latest") {
                NSWorkspace.shared.open(release)
            }
        } catch { updateStatus="Update check failed: \(error.localizedDescription)" }
    }
    func detectGame() {
        let matches = NSWorkspace.shared.runningApplications.filter { app in
            let name = (app.localizedName ?? "").lowercased()
            return name.contains("the sims 4") || name.contains("sims 4")
        }
        casStatus = matches.isEmpty ? "The Sims 4 is not running." : "The Sims 4 process detected. Enter CAS before capturing."
    }
    func captureGame() {
        let matches = NSWorkspace.shared.runningApplications.filter { app in
            let name = (app.localizedName ?? "").lowercased()
            return name.contains("the sims 4") || name.contains("sims 4")
        }
        guard !matches.isEmpty else { casStatus = "The Sims 4 is not running."; return }
        let picker = NSSavePanel()
        picker.nameFieldStringValue = "Simcraft-CAS-Capture.png"
        guard picker.runModal() == .OK, let url = picker.url else { return }
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/sbin/screencapture")
        task.arguments = ["-x", url.path]
        do {
            try task.run()
            task.waitUntilExit()
            casStatus = task.terminationStatus == 0 ? "Screen captured. CAS mode is not automatically verified." : "Capture failed. Check macOS Screen Recording permissions."
        } catch { casStatus = "Capture failed: \(error.localizedDescription)" }
    }
    func exportOutfit() {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "simcraft-outfit.json"
        guard panel.runModal() == .OK, let url = panel.url else { return }
        let payload: [String: Any] = ["occasion": occasion, "styles": Array(styles).sorted()]
        do {
            let data = try JSONSerialization.data(withJSONObject: payload, options: [.prettyPrinted, .sortedKeys])
            try data.write(to: url)
        } catch { message = "Outfit export failed: \(error.localizedDescription)" }
    }
    func exportProject() {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "mod.simcraft.json"
        guard panel.runModal() == .OK, let url = panel.url else { return }
        let data: [String: Any] = ["format":"simcraft-mod-project","schemaVersion":1,"name":modName,"category":category,"description":details,"compatibilityChecks":compatibility,"output":"specification-only"]
        do {
            let bytes = try JSONSerialization.data(withJSONObject: data, options: [.prettyPrinted, .sortedKeys])
            try bytes.write(to: url)
            message = "Project specification exported (not an installable mod)."
        } catch { message = "Export failed: \(error.localizedDescription)" }
    }
}
