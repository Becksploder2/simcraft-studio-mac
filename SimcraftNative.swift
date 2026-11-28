import SwiftUI
import AppKit
import UniformTypeIdentifiers
import Vision
import ImageIO

@main struct SimcraftNative: App {
    var body: some Scene {
        WindowGroup("Simcraft Studio") { StudioView().frame(minWidth: 950, minHeight: 650) }
    }
}
struct StudioView: View {
    let tabs = ["Simcraft"]
    @State private var selected = "Simcraft"
    @State private var modName = ""
    @State private var category = "Gameplay interactions"
    @State private var details = ""
    @State private var compatibility = true
    @State private var photoURL: URL?
    @State private var photoGeneration = 0
    @State private var photoPreview: NSImage?
    @State private var faceStatus = "Face recognition: no photo analyzed"
    @State private var analyzingFace = false
    @State private var smartAdapt = true
    @State private var templateGender = "F"
    @State private var templateNumber = 1
    @State private var glasses = "Yes"
    @State private var hair = "Match reference"
    @State private var occasion = "Everyday"
    @State private var styles: Set<String> = []
    @State private var buildDescription = ""
    @State private var buildWidth = 30
    @State private var buildDepth = 20
    @State private var buildStatus = "Not connected to Build Mode."
    @State private var buildDraft = ""
    @State private var packageReport = "No package imported."
    @State private var catalogStatus = "No catalog imported"
    @State private var catalogPreview = ""
    @State private var casStatus = "CAS not checked"
    @State private var casCaptureURL: URL?
    @State private var backgroundURL: URL?
    @State private var updateStatus = "Update status: not checked"
    @State private var message = "Project specification only; not an installable mod."
    var body: some View {
        VStack(spacing: 0) {
            ScrollView(.horizontal, showsIndicators: true) {
                HStack(spacing: 2) {
                    ForEach(tabs, id: \.self) { item in
                        Button(item) { selected = item }
                            .buttonStyle(.bordered)
                            .tint(selected == item ? .accentColor : .gray)
                            .controlSize(.small)
                    }
                }.padding(.horizontal, 6).padding(.vertical, 5)
            }
            ZStack {
                if let url = Bundle.main.url(forResource: "Default CAS Room", withExtension: "png", subdirectory: "Backgrounds"),
                   let room = NSImage(contentsOf: url) {
                    GeometryReader { geometry in
                        Image(nsImage: room).resizable().scaledToFill()
                            .frame(width: geometry.size.width, height: geometry.size.height)
                            .clipped()
                    }.ignoresSafeArea()
                    Color.black.opacity(0.35).ignoresSafeArea()
                }
            ScrollView(.vertical) {
            VStack(alignment: .leading, spacing: 8) {
                Text(selected == "Simcraft" ? "Simcraft — Sim Creator" : selected).font(.headline).bold()
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
                } else if selected == "Simcraft" {
                    Button("Import reference photo") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [.png, .jpeg, .bmp]
                        if picker.runModal() == .OK, let url = picker.url { importPhoto(url) }
                    }
                    Text(photoURL == nil ? "Ready to import a reference photo" : "Reference photo imported")
                    ZStack {
                        Color(red: 0.13, green: 0.17, blue: 0.25)
                        if let image = photoPreview { Image(nsImage: image).resizable().scaledToFit() }
                    }.frame(width: 420, height: 330)
                    Button(analyzingFace ? "Analyzing..." : "Analyze face landmarks") { analyzeFace() }.disabled(photoURL == nil || analyzingFace)
                    Text(faceStatus).foregroundStyle(.secondary)
                    Picker("Template group", selection: $templateGender) {
                        Text("Female").tag("F"); Text("Male").tag("M")
                    }
                    Toggle("SmartAdapt: automatically match facial template", isOn: $smartAdapt)
                    Picker("Template selection", selection: $templateNumber) {
                        ForEach(1...19, id: \.self) { n in Text("\(templateGender)\(n)").tag(n) }
                    }
                    if let url = Bundle.main.url(forResource: "\(templateGender)\(templateNumber)", withExtension: "png", subdirectory: "Templates/\(templateGender == "F" ? "Female" : "Male")"), let templateImage = NSImage(contentsOf: url) {
                        Image(nsImage: templateImage).resizable().scaledToFit().frame(width: 220, height: 180)
                    }
                    Text("SIM MORPH STUDIO — ORIGINAL CONTROLS").font(.headline)
                    Button("Refresh Preview") { casStatus = "Preview refreshed." }
                    Button("Set CAS Click Point") { casStatus = "CAS click calibration requires a game connection; no click sent." }
                    Button("Apply One CAS Click") { casStatus = "No CAS click sent: calibration and game control are not available." }
                    Button("Detect The Sims 4") { detectGame() }
                    Button("Capture Current CAS Face") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [.png, .jpeg]
                        picker.begin { response in
                            guard response == .OK, let url = picker.url else { return }
                            casStatus = "CAS screenshot selected: \\(url.lastPathComponent). Mode not verified."
                        }
                    }
                    Text("Or drop a screenshot here (no file picker)")
                        .padding(12).frame(maxWidth: 520).overlay(RoundedRectangle(cornerRadius: 6).stroke(.gray))
                        .onDrop(of: [UTType.fileURL.identifier], isTargeted: nil) { providers in
                            guard let provider = providers.first else { return false }
                            _ = provider.loadObject(ofClass: URL.self) { url, _ in
                                guard let url = url, ["png","jpg","jpeg"].contains(url.pathExtension.lowercased()) else { return }
                                DispatchQueue.main.async { casStatus = "CAS screenshot selected: \\(url.lastPathComponent). Mode not verified." }
                            }
                            return true
                        }
                    Text(casStatus).foregroundStyle(.secondary)
                    Text("CAS workspace background (local preview)")
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
                } else if selected == "Buildcraft" {
                    Text("Buildcraft — Sims 4 Build Mode workspace (early prototype)").font(.headline)
                    Text("Describe the building or lot")
                    TextEditor(text: $buildDescription).frame(height: 110).border(.gray)
                    Stepper("Lot width: \\(buildWidth) tiles", value: $buildWidth, in: 10...64)
                    Stepper("Lot depth: \\(buildDepth) tiles", value: $buildDepth, in: 10...64)
                    Button("Detect The Sims 4") {
                        detectGame()
                        buildStatus = casStatus + " Build Mode is not verified."
                    }
                    Text(buildStatus)
                    Button("Import Build/Buy screenshot") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [.png, .jpeg]
                        picker.begin { response in
                            guard response == .OK, let url = picker.url else { return }
                            buildStatus = "Build/Buy screenshot selected: \\(url.lastPathComponent). Mode not verified."
                        }
                    }
                    Text("Or drop a Build/Buy screenshot here (no file picker)")
                        .padding(12).frame(maxWidth: 520).overlay(RoundedRectangle(cornerRadius: 6).stroke(.gray))
                        .onDrop(of: [UTType.fileURL.identifier], isTargeted: nil) { providers in
                            guard let provider = providers.first else { return false }
                            _ = provider.loadObject(ofClass: URL.self) { url, _ in
                                guard let url = url, ["png","jpg","jpeg"].contains(url.pathExtension.lowercased()) else { return }
                                DispatchQueue.main.async { buildStatus = "Build/Buy screenshot selected: \\(url.lastPathComponent). Mode not verified." }
                            }
                            return true
                        }
                    Button("Create starter build plan") {
                        guard !buildDescription.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
                            buildStatus = "Describe the building first."; return
                        }
                        buildDraft = "BUILDCRAFT DRAFT PLAN\\nLot: \\(buildWidth) x \\(buildDepth) tiles\\nDescription: \\(buildDescription)\\n\\nDraft specification only. AI generation and construction are not connected."
                        buildStatus = "Draft created locally. No game actions performed."
                    }
                    Text(buildDraft).textSelection(.enabled)
                    Button("Export Buildcraft project") { exportBuildcraft() }
                    Text("No automated game input. Build Mode must be verified before construction.").foregroundStyle(.secondary)
                } else if selected == "CAS Mod Studio" {
                    Text("CAS Mod Studio — inspect existing Sims 4 packages (read-only)")
                    Button("Import .package reference") {
                        let picker = NSOpenPanel()
                        picker.allowedContentTypes = [UTType(filenameExtension: "package") ?? .data]
                        if picker.runModal() == .OK, let url = picker.url { inspectPackage(url) }
                    }
                    ScrollView { Text(packageReport).font(.system(.body, design: .monospaced)).textSelection(.enabled) }
                    Text("Original packages are not modified. Resource extraction and editing are not yet implemented.").foregroundStyle(.secondary)
                } else if selected == "Settings" {
                    Text("Native desktop preview. The HTML workspace is not used by this interface.")
                    Button("Check for updates") { Task { await checkForUpdates() } }
                    Text(updateStatus).foregroundStyle(.secondary)
                    Text("Photo import, manual template selection, preferences, outfit plans, CC catalog import and AI mod specifications are available.")
                } else {
                    Text("Native migration in progress. The previous HTML tool has not yet been ported.").foregroundStyle(.secondary)
                }
                Spacer()
            }.padding(24).frame(maxWidth: .infinity, alignment: .topLeading)
            }
            }.frame(maxWidth: .infinity, maxHeight: .infinity)
        }.frame(minWidth: 950, minHeight: 650)
         .task { await checkForUpdates() }
         .task { while !Task.isCancelled { detectGame(); buildStatus = casStatus + " Build/Buy mode not yet verified."; try? await Task.sleep(for: .seconds(3)) } }
    }
    @MainActor func checkForUpdates() async {
        updateStatus = "Checking GitHub releases..."
        guard let url=URL(string:"https://api.github.com/repos/Becksploder2/simcraft-studio-mac/releases/latest") else { return }
        do {
            var request=URLRequest(url:url)
            request.setValue("SimcraftStudio/" + (Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0"), forHTTPHeaderField:"User-Agent")
            let (data,_) = try await URLSession.shared.data(for:request)
            guard let json=try JSONSerialization.jsonObject(with:data) as? [String:Any],
                  let tag=json["tag_name"] as? String else { updateStatus="Invalid release information"; return }
            let latest=tag.trimmingCharacters(in:CharacterSet(charactersIn:"vV"))
            let installed = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0"
            if latest.compare(installed,options:.numeric) != .orderedDescending {
                updateStatus="Up to date (v" + installed + ")."; return
            }
            updateStatus="Update available: \(tag)"
            let alert=NSAlert()
            alert.messageText="Simcraft Studio \(tag) is available"
            alert.informativeText="Download the official Mac update to Downloads? Installation is manual."
            alert.addButton(withTitle:"Download update")
            alert.addButton(withTitle:"Later")
            if alert.runModal() == .alertFirstButtonReturn {
                let assets=json["assets"] as? [[String:Any]] ?? []
                let asset=assets.first { item in
                    let n=(item["name"] as? String ?? "").lowercased()
                    return n.hasSuffix(".dmg") || (n.hasSuffix(".zip") && n.contains("mac"))
                }
                if let link=asset?["browser_download_url"] as? String, let downloadURL=URL(string:link) {
                    updateStatus="Downloading update..."
                    let (temp,response)=try await URLSession.shared.download(from:downloadURL)
                    guard let http=response as? HTTPURLResponse, http.statusCode==200 else {
                        updateStatus="Update download failed."; return
                    }
                    let downloads=FileManager.default.urls(for:.downloadsDirectory,in:.userDomainMask)[0]
                    let filename=tag+"-"+(asset?["name"] as? String ?? "Simcraft-update.zip")
                    let destination=downloads.appendingPathComponent(filename)
                    if FileManager.default.fileExists(atPath:destination.path) {
                        try FileManager.default.removeItem(at:destination)
                    }
                    try FileManager.default.moveItem(at:temp,to:destination)
                    updateStatus="Update downloaded: "+filename
                    NSWorkspace.shared.activateFileViewerSelecting([destination])
                } else if let release=URL(string:"https://github.com/Becksploder2/simcraft-studio-mac/releases/latest") {
                    NSWorkspace.shared.open(release)
                }
            }
        } catch { updateStatus="Update check failed: \(error.localizedDescription)" }
    }
    @MainActor func importPhoto(_ url: URL) {
        photoGeneration += 1
        let generation = photoGeneration
        analyzingFace = false
        templateNumber = 1
        casCaptureURL = nil
        casStatus = "New photo: morph session reset. Previous in-game face changes must be reset manually."
        photoURL = url
        photoPreview = nil
        faceStatus = "Loading photo..."
        Task {
            let image: NSImage? = await Task.detached(priority: .userInitiated) {
                guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
                      let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any],
                      let width = properties[kCGImagePropertyPixelWidth] as? Int,
                      let height = properties[kCGImagePropertyPixelHeight] as? Int,
                      width > 0, height > 0, width <= 8192, height <= 8192,
                      Int64(width) * Int64(height) <= 30_000_000,
                      let cg = CGImageSourceCreateThumbnailAtIndex(source, 0, [
                        kCGImageSourceCreateThumbnailFromImageAlways: true,
                        kCGImageSourceThumbnailMaxPixelSize: 1200
                      ] as CFDictionary) else { return nil }
                return NSImage(cgImage: cg, size: NSSize(width: cg.width, height: cg.height))
            }.value
            guard generation == photoGeneration else { return }
            photoPreview = image
            faceStatus = image == nil ? "Photo unavailable or too large." : "Photo loaded. Analyzing..."
            if image != nil { analyzeFace() }
        }
    }
    @MainActor func analyzeFace() {
        guard let url = photoURL, !analyzingFace else { return }
        let generation = photoGeneration
        let group = templateGender
        let automatic = smartAdapt
        let templates: [(Int, URL)] = (1...19).compactMap { n in
            guard let file = Bundle.main.url(forResource: group + String(n), withExtension: "png", subdirectory: "Templates/" + (group == "F" ? "Female" : "Male")) else { return nil }
            return (n, file)
        }
        analyzingFace = true
        faceStatus = "Analyzing face..."
        Task {
            let result: (String, Int?) = await Task.detached(priority: .userInitiated) {
                func signature(_ file: URL) throws -> [Double]? {
                    let request = VNDetectFaceLandmarksRequest()
                    try VNImageRequestHandler(url: file, options: [:]).perform([request])
                    guard let faces = request.results, faces.count == 1, let lm = faces[0].landmarks else { return nil }
                    let regions = [lm.leftEye, lm.rightEye, lm.nose, lm.outerLips, lm.faceContour]
                    var values: [Double] = []
                    for region in regions {
                        guard let points = region?.normalizedPoints, !points.isEmpty else { return nil }
                        let xs = points.map { Double($0.x) }
                        let ys = points.map { Double($0.y) }
                        values += [xs.reduce(0,+)/Double(xs.count), ys.reduce(0,+)/Double(ys.count),
                                   (xs.max() ?? 0)-(xs.min() ?? 0), (ys.max() ?? 0)-(ys.min() ?? 0)]
                    }
                    return values
                }
                do {
                    guard let reference = try signature(url) else { return ("Use a clear photo containing one recognizable face.", nil) }
                    guard automatic else { return ("Face landmarks recognized. Manual template selection enabled.", nil) }
                    var best: (Int, Double)?
                    for (n, file) in templates {
                        guard let candidate = try? signature(file), candidate.count == reference.count else { continue }
                        let score = zip(reference, candidate).reduce(0.0) { total, pair in
                            let d = pair.0 - pair.1
                            return total + d*d
                        }
                        if best == nil || score < best!.1 { best = (n, score) }
                    }
                    guard let match = best else { return ("No template could be analyzed. Select manually.", nil) }
                    return ("SmartAdapt suggested " + group + String(match.0) + ". Verify the estimated match in CAS.", match.0)
                } catch { return ("Face analysis failed: " + error.localizedDescription, nil) }
            }.value
            guard generation == photoGeneration else { return }
            analyzingFace = false
            guard group == templateGender else { faceStatus = "Template group changed. Analyze again."; return }
            faceStatus = result.0
            if smartAdapt, let match = result.1 { templateNumber = match }
        }
    }
    @MainActor func inspectPackage(_ url: URL) {
        packageReport = "Inspecting package..."
        Task.detached(priority: .userInitiated) {
            let result: String
            do {
                let handle = try FileHandle(forReadingFrom: url)
                defer { try? handle.close() }
                let header = try handle.read(upToCount: 96) ?? Data()
                guard header.count >= 96, String(data: header.prefix(4), encoding: .ascii) == "DBPF" else {
                    throw NSError(domain: "Simcraft", code: 1, userInfo: [NSLocalizedDescriptionKey: "Not a valid DBPF package header."])
                }
                let bytes = Array(header)
                let count = UInt32(bytes[36]) | UInt32(bytes[37]) << 8 | UInt32(bytes[38]) << 16 | UInt32(bytes[39]) << 24
                guard count <= 2_000_000 else {
                    throw NSError(domain: "Simcraft", code: 2, userInfo: [NSLocalizedDescriptionKey: "Resource count exceeds safety limit."])
                }
                let size = (try FileManager.default.attributesOfItem(atPath: url.path)[.size] as? NSNumber)?.int64Value ?? 0
                result = "File: \\(url.lastPathComponent)\\nSize: \\(size) bytes\\nFormat: DBPF\\nHeader resource count: \\(count)\\n\\nRead-only reference inspection. Resource extraction and editing are not yet implemented."
            } catch { result = "Package inspection failed: \\(error.localizedDescription)" }
            await MainActor.run { packageReport = result }
        }
    }
    func captureBuildMode() {
        let running = NSWorkspace.shared.runningApplications.contains {
            let name = ($0.localizedName ?? "").lowercased()
            return name.contains("the sims 4") || name.contains("sims 4")
        }
        guard running else { buildStatus = "The Sims 4 is not running."; return }
        let picker = NSSavePanel()
        picker.nameFieldStringValue = "Simcraft-BuildBuy-Capture.png"
        guard picker.runModal() == .OK, let url = picker.url else { return }
        let task = Process()
        task.executableURL = URL(fileURLWithPath: "/usr/sbin/screencapture")
        task.arguments = ["-x", url.path]
        do {
            try task.run()
            task.terminationHandler = { finished in
                DispatchQueue.main.async {
                    buildStatus = finished.terminationStatus == 0
                        ? "Screenshot saved. Build/Buy mode not yet verified."
                        : "Capture failed. Check Screen Recording permissions."
                }
            }
            buildStatus = "Capturing screen..."
        } catch { buildStatus = "Capture failed: \\(error.localizedDescription)" }
    }
    func exportBuildcraft() {
        guard !buildDescription.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            buildStatus = "Describe the building first."; return
        }
        let picker = NSSavePanel()
        picker.nameFieldStringValue = "buildcraft-project.json"
        guard picker.runModal() == .OK, let url = picker.url else { return }
        do {
            let data = try JSONSerialization.data(withJSONObject: [
                "format": "simcraft-buildcraft-project",
                "schemaVersion": 1,
                "lotWidth": buildWidth,
                "lotDepth": buildDepth,
                "description": buildDescription,
                "constructionEnabled": false
            ], options: [.prettyPrinted, .sortedKeys])
            try data.write(to: url)
            buildStatus = "Buildcraft project exported."
        } catch { buildStatus = "Export failed: \\(error.localizedDescription)" }
    }
    func detectGame() {
        let matches = NSWorkspace.shared.runningApplications.filter { app in
            let name = (app.localizedName ?? "").lowercased()
            return name.contains("the sims 4") || name.contains("sims 4")
        }
        casStatus = matches.isEmpty ? "The Sims 4 is not running." : "The Sims 4 is running. CAS mode cannot be verified from process detection alone."
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
            task.terminationHandler = { finished in
                DispatchQueue.main.async {
                    casStatus = finished.terminationStatus == 0 ? "Screen captured. CAS mode is not automatically verified." : "Capture failed. Check macOS Screen Recording permissions."
                }
            }
            casStatus = "Capturing screen..."
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
