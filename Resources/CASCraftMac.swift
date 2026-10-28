import Cocoa
import WebKit

final class Studio: NSObject, NSApplicationDelegate {
    var window: NSWindow!
    var embeddedWorkspace: WKWebView?
    var reference = NSImageView()
    var cas = NSImageView()
    var backdrop = NSImageView()
    var status = NSTextField(labelWithString: "Ready — Mac test edition (no automatic morphing)")
    var templateSelector = NSPopUpButton()
    var backgroundSelector = NSPopUpButton()
    var root: URL = URL(fileURLWithPath: CommandLine.arguments[0]).deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent().appendingPathComponent("Resources")
    var selectedReference: URL?
    let studioCategories = ["Face and skin","Eyes and eyebrows","Nose, cheeks and jaw","Mouth and teeth","Glasses and accessories","Hair and facial hair","Makeup and skin details","Neck and body","Tops and bottoms","Full-body outfits","Shoes and socks","Hats and jewelry","Outfit categories and colors"]
    let studioGuides = [
      "Match face preset and skin details in CAS.",
      "Compare eye spacing, eyelids and brows in Detail Edit Mode.",
      "Inspect bridge, nose tip, cheeks, jaw and chin.",
      "Select available mouth and teeth presets; braces may require custom content.",
      "Match glasses frame, lenses and accessories from installed CAS items.",
      "Match hairstyle, parting, length and color.",
      "Match visible makeup, freckles and skin details.",
      "Compare neck and body silhouette from multiple angles.",
      "Choose tops and bottoms from installed CAS content.",
      "Choose available full-body outfits.",
      "Match shoes, socks and legwear.",
      "Choose hats, earrings, necklaces and jewelry.",
      "Build individual Everyday, Formal, Athletic, Sleep and other outfits."
    ]

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        let menu = NSMenu(); let appItem = NSMenuItem(); menu.addItem(appItem)
        let appMenu = NSMenu(); appMenu.addItem(withTitle: "Quit CASCraft Studio", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        appItem.submenu = appMenu
        let studioItem = NSMenuItem(title:"Complete CAS Studio", action:nil, keyEquivalent:"")
        let studioMenu = NSMenu(title:"Complete CAS Studio")
        for (index,name) in studioCategories.enumerated() {
            let item = NSMenuItem(title:name+" (guide)…", action:#selector(showStudioGuide(_:)), keyEquivalent:"")
            item.target=self;item.tag=index;studioMenu.addItem(item)
        }
        studioMenu.addItem(.separator())
        let exportItem = NSMenuItem(title:"Export Complete CAS Plan", action:#selector(exportStudioPlan), keyEquivalent:"")
        exportItem.target=self;studioMenu.addItem(exportItem)
        studioItem.submenu=studioMenu;menu.addItem(studioItem)
        let creatorItem = NSMenuItem(title:"CASCraft Creator (Prototype)", action:nil, keyEquivalent:"")
        let creatorMenu = NSMenu(title:"CASCraft Creator")
        let glassesItem = NSMenuItem(title:"Open Glasses Creator…", action:#selector(openGlassesCreator), keyEquivalent:"")
        glassesItem.target=self;creatorMenu.addItem(glassesItem);creatorItem.submenu=creatorMenu;menu.addItem(creatorItem)
        let assistItem=NSMenuItem(title:"SmartAssist / Mod Studio",action:#selector(openSmartAssist),keyEquivalent:"");assistItem.target=self;menu.addItem(assistItem)
        let morphItem=NSMenuItem(title:"SmartMorph / Photo Studio",action:#selector(openSmartMorph),keyEquivalent:"");morphItem.target=self;menu.addItem(morphItem)
        let textureItem=NSMenuItem(title:"Face Texture Lab v4.8",action:#selector(openFaceTextureLab),keyEquivalent:"");textureItem.target=self;menu.addItem(textureItem)
        let uvItem=NSMenuItem(title:"UV Face Alignment v4.9",action:#selector(openUVFaceAlign),keyEquivalent:"");uvItem.target=self;menu.addItem(uvItem)
        let tabsItem=NSMenuItem(title:"Simscraft Studio 5.0 Tabs",action:#selector(openStudioTabs),keyEquivalent:"t");tabsItem.target=self;menu.addItem(tabsItem)
        NSApp.mainMenu = menu
        window = NSWindow(contentRect: NSRect(x:0,y:0,width:1050,height:690), styleMask:[.titled,.closable,.miniaturizable,.resizable], backing:.buffered, defer:false)
        window.title = "CASCraft Studio — Mac Test Edition v4.7"
        window.center(); window.minSize = NSSize(width:850,height:570)
        guard let content = window.contentView else { return }
        backdrop.imageScaling = .scaleAxesIndependently; backdrop.image = NSImage(contentsOf:root.appendingPathComponent("Backgrounds/Default CAS Room.png"))
        backdrop.frame = content.bounds; backdrop.autoresizingMask = [.width,.height]; content.addSubview(backdrop)
        let veil = NSView(frame:content.bounds); veil.wantsLayer=true;veil.layer?.backgroundColor=NSColor.black.withAlphaComponent(0.67).cgColor
        veil.autoresizingMask=[.width,.height];content.addSubview(veil)
        let heading = NSTextField(labelWithString:"CASCraft Studio — Mac Test Edition")
        heading.font = NSFont.boldSystemFont(ofSize:23); heading.textColor = .white
        heading.frame=NSRect(x:24,y:635,width:700,height:34); heading.autoresizingMask=[.minYMargin];content.addSubview(heading)
        reference = imageView(x:25,y:220,w:290,h:385,into:content)
        cas = imageView(x:340,y:220,w:685,h:385,into:content)
        addLabel("Reference photo",x:25,y:608,into:content);addLabel("Current CAS screenshot",x:340,y:608,into:content)
        button("Import Reference Photo",x:25,y:165,w:220,action:#selector(importPhoto),into:content)
        button("Capture CAS (select area)",x:340,y:165,w:230,action:#selector(captureCAS),into:content)
        button("Open Screenshot",x:580,y:165,w:170,action:#selector(openCAS),into:content)
        addLabel("Face template",x:25,y:125,into:content)
        templateSelector.frame=NSRect(x:25,y:89,width:235,height:28);templateSelector.target=self;templateSelector.action=#selector(selectTemplate)
        templateSelector.addItems(withTitles:["Select a template"] + (1...19).map{"Female F\($0)"} + (1...19).map{"Male M\($0)"});content.addSubview(templateSelector)
        button("Preview Template",x:270,y:89,w:160,action:#selector(selectTemplate),into:content)
        addLabel("Background theme",x:450,y:125,into:content)
        backgroundSelector.frame=NSRect(x:450,y:89,width:240,height:28);backgroundSelector.target=self;backgroundSelector.action=#selector(selectBackground);content.addSubview(backgroundSelector)
        button("Add Background",x:700,y:89,w:160,action:#selector(importBackground),into:content)
        refreshBackgrounds()
        status.frame=NSRect(x:25,y:27,width:990,height:44);status.textColor=.white;status.font=NSFont.systemFont(ofSize:13)
        status.lineBreakMode = .byWordWrapping;status.maximumNumberOfLines=2;content.addSubview(status)
        showEmbeddedWorkspace()
        window.makeKeyAndOrderFront(nil);NSApp.activate(ignoringOtherApps:true)
    }
    @objc func showStudioGuide(_ sender:NSMenuItem) {
        let index=sender.tag
        guard index >= 0 && index < studioCategories.count else { return }
        let alert=NSAlert()
        alert.messageText="Complete CAS Studio — "+studioCategories[index]
        alert.informativeText=studioGuides[index]+"\\n\\n1. Import a reference photo.\\n2. Capture the CAS view.\\n3. Edit the corresponding category in The Sims 4.\\n4. Recapture and compare.\\n\\nAutomatic catalog selection and body sculpting are not implemented."
        alert.runModal()
        status.stringValue="CAS Studio guide: "+studioCategories[index]
    }
    @objc func exportStudioPlan() {
        let panel=NSSavePanel()
        panel.nameFieldStringValue="SimMorph_Complete_CAS_Plan.json"
        guard panel.runModal() == .OK, let url=panel.url else {return}
        let payload:[String:Any]=["version":"4.4","mode":"complete_cas_studio","categories":studioCategories,"guides":studioGuides,"reference_photo":selectedReference?.path ?? "","limitations":"Guided manual CAS workflow; no automatic catalog selection or body sculpting."]
        do {let data=try JSONSerialization.data(withJSONObject:payload,options:[.prettyPrinted,.sortedKeys]);try data.write(to:url);status.stringValue="CAS plan saved: "+url.lastPathComponent}
        catch {status.stringValue="Plan export failed: "+error.localizedDescription}
    }
    func imageView(x:CGFloat,y:CGFloat,w:CGFloat,h:CGFloat,into v:NSView)->NSImageView {
        let iv=NSImageView(frame:NSRect(x:x,y:y,width:w,height:h));iv.imageScaling = .scaleProportionallyUpOrDown
        iv.wantsLayer=true;iv.layer?.backgroundColor=NSColor.black.withAlphaComponent(0.65).cgColor
        iv.layer?.borderColor=NSColor.white.withAlphaComponent(0.4).cgColor;iv.layer?.borderWidth=1
        v.addSubview(iv);return iv
    }
    func addLabel(_ s:String,x:CGFloat,y:CGFloat,into v:NSView){let t=NSTextField(labelWithString:s);t.textColor = .white;t.font=NSFont.boldSystemFont(ofSize:13);t.frame=NSRect(x:x,y:y,width:320,height:20);v.addSubview(t)}
    func button(_ s:String,x:CGFloat,y:CGFloat,w:CGFloat,action:Selector,into v:NSView){let b=NSButton(title:s,target:self,action:action);b.frame=NSRect(x:x,y:y,width:w,height:32);b.bezelStyle = .rounded;v.addSubview(b)}
    func pickFile(types:[String])->URL?{let p=NSOpenPanel();p.allowedFileTypes=types;p.allowsMultipleSelection=false;return p.runModal() == .OK ? p.url : nil}
    func showEmbeddedWorkspace() {
        let resources = Bundle.main.resourceURL ?? root
        let file = resources.appendingPathComponent("Studio.html")
        guard FileManager.default.fileExists(atPath:file.path) else {
            status.stringValue = "Studio.html missing from Resources"
            return
        }
        let configuration = WKWebViewConfiguration()
        let web = WKWebView(frame: window.contentView?.bounds ?? .zero, configuration: configuration)
        web.autoresizingMask = [.width, .height]
        window.contentView = web
        window.title = "Simscraft Studio — CAS Workspace"
        window.setContentSize(NSSize(width: 1350, height: 850))
        window.minSize = NSSize(width: 950, height: 640)
        web.loadFileURL(file, allowingReadAccessTo: resources)
        embeddedWorkspace = web
    }
    @objc func openStudioTabs() {
        showEmbeddedWorkspace()
    }
    @objc func openUVFaceAlign() {
        let executableFolder=Bundle.main.executableURL!.deletingLastPathComponent()
        let root=Bundle.main.resourceURL ?? executableFolder
        let options=[executableFolder.appendingPathComponent("UVFaceAlign.html"),root.appendingPathComponent("UVFaceAlign.html")]
        if let file=options.first(where:{FileManager.default.fileExists(atPath:$0.path)}) { NSWorkspace.shared.open(file) }
        else { status.stringValue="UVFaceAlign.html missing in Resources" }
    }
    @objc func openFaceTextureLab() {
        let root=Bundle.main.resourceURL ?? URL(fileURLWithPath:FileManager.default.currentDirectoryPath)
        let executableFolder=Bundle.main.executableURL!.deletingLastPathComponent()
        let options=[executableFolder.appendingPathComponent("FaceTextureLab.html"),root.appendingPathComponent("FaceTextureLab.html")]
        if let url=options.first(where:{FileManager.default.fileExists(atPath:$0.path)}) { NSWorkspace.shared.open(url); status.stringValue="Face Texture Lab opened" }
        else { status.stringValue="FaceTextureLab.html missing in Resources" }
    }
    @objc func openSmartMorph() {
        let executableFolder = URL(fileURLWithPath: CommandLine.arguments[0]).deletingLastPathComponent()
        let options = [executableFolder.appendingPathComponent("SmartMorph.html"),root.appendingPathComponent("SmartMorph.html")]
        if let url = options.first(where: { FileManager.default.fileExists(atPath:$0.path) }) { NSWorkspace.shared.open(url) }
        else { status.stringValue = "SmartMorph.html missing in Resources" }
    }
    @objc func openSmartAssist() {
        let executableFolder = URL(fileURLWithPath: CommandLine.arguments[0]).deletingLastPathComponent()
        let options = [executableFolder.appendingPathComponent("SmartAssist.html"),root.appendingPathComponent("SmartAssist.html")]
        if let url = options.first(where: { FileManager.default.fileExists(atPath:$0.path) }) { NSWorkspace.shared.open(url) }
        else { status.stringValue = "SmartAssist.html not found in app resources" }
    }
    @objc func openGlassesCreator() {
        let executableFolder = URL(fileURLWithPath: CommandLine.arguments[0]).deletingLastPathComponent()
        let options = [executableFolder.appendingPathComponent("GlassesCreator.html"),root.appendingPathComponent("GlassesCreator.html")]
        if let url = options.first(where: { FileManager.default.fileExists(atPath:$0.path) }) { NSWorkspace.shared.open(url) }
        else { status.stringValue = "GlassesCreator.html not found beside app or in Resources" }
    }
    @objc func importPhoto(){guard let u=pickFile(types:["png","jpg","jpeg","heic"]) else{return};guard let im=NSImage(contentsOf:u) else{status.stringValue="Could not open image";return};selectedReference=u;reference.image=im;status.stringValue="Reference loaded: \(u.lastPathComponent)"}
    @objc func openCAS(){guard let u=pickFile(types:["png","jpg","jpeg"]) else{return};cas.image=NSImage(contentsOf:u);status.stringValue="CAS screenshot opened: \(u.lastPathComponent)"}
    @objc func captureCAS(){
        let out=FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Pictures/SimMorph_CAS_Capture.png")
        let proc=Process();proc.executableURL=URL(fileURLWithPath:"/usr/sbin/screencapture");proc.arguments=["-i",out.path]
        do {try proc.run();proc.waitUntilExit();if proc.terminationStatus == 0,let im=NSImage(contentsOf:out){cas.image=im;status.stringValue="CAS captured: \(out.path)"}else{status.stringValue="Capture canceled or failed. Allow Screen Recording permission in macOS Settings."}}
        catch {status.stringValue="Screenshot failed: \(error.localizedDescription)"}
    }
    @objc func selectTemplate(){let s=templateSelector.titleOfSelectedItem ?? "";let parts=s.split(separator:" ");guard parts.count == 2 else{return};let id=String(parts[1]);let gender=parts[0] == "Female" ? "Female":"Male"
        let u=root.appendingPathComponent("Templates/\(gender)/\(id).png")
        if let im=NSImage(contentsOf:u){cas.image=im;status.stringValue="Showing template \(id) for reference only. This does NOT modify the Sim in-game."}else{status.stringValue="Template image missing: \(id)"}}
    func refreshBackgrounds(){backgroundSelector.removeAllItems();let folder=root.appendingPathComponent("Backgrounds")
        let files=(try? FileManager.default.contentsOfDirectory(at:folder,includingPropertiesForKeys:nil)) ?? []
        let images=files.filter{["png","jpg","jpeg"].contains($0.pathExtension.lowercased())}.sorted{$0.lastPathComponent < $1.lastPathComponent}
        backgroundSelector.addItems(withTitles:images.map{$0.lastPathComponent})}
    @objc func selectBackground(){guard let name=backgroundSelector.titleOfSelectedItem else{return};let u=root.appendingPathComponent("Backgrounds").appendingPathComponent(name);backdrop.image=NSImage(contentsOf:u);status.stringValue="Background: \(name)"}
    @objc func importBackground(){guard let u=pickFile(types:["png","jpg","jpeg"]) else{return};let dest=root.appendingPathComponent("Backgrounds").appendingPathComponent(u.lastPathComponent)
        do{if u != dest {if FileManager.default.fileExists(atPath:dest.path){try FileManager.default.removeItem(at:dest)};try FileManager.default.copyItem(at:u,to:dest)};refreshBackgrounds();backgroundSelector.selectItem(withTitle:dest.lastPathComponent);selectBackground()}
        catch{status.stringValue="Couldn't import background: \(error.localizedDescription)"}}
}
let app=NSApplication.shared;let delegate=Studio();app.delegate=delegate;app.run()
