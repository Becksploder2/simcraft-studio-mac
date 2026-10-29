import SwiftUI
import AppKit

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
    @State private var message = "Project specification only; not an installable mod."
    var body: some View {
        NavigationSplitView {
            List(tabs, id: \.self, selection: $selected) { Text($0) }
                .navigationTitle("Simcraft Studio")
                .frame(minWidth: 220)
        } detail: {
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
                } else {
                    Text("Native migration in progress. The previous HTML tool has not yet been ported.").foregroundStyle(.secondary)
                }
                Spacer()
            }.padding(28).frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
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
