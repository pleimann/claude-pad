import AppKit
import WebKit
import Foundation

// ─── IPC helpers ─────────────────────────────────────────────────────────────

func emit(_ obj: [String: Any]) {
    guard let data = try? JSONSerialization.data(withJSONObject: obj),
          let line = String(data: data, encoding: .utf8) else { return }
    print(line)
    fflush(stdout)
}

// ─── Popover content view controller ─────────────────────────────────────────

class PopoverWebViewController: NSViewController, WKScriptMessageHandler {
    var webView: WKWebView!
    var onClose: (() -> Void)?

    init(url: URL, width: CGFloat, height: CGFloat, onClose: @escaping () -> Void) {
        super.init(nibName: nil, bundle: nil)
        self.onClose = onClose

        let config = WKWebViewConfiguration()
        // Override window.close() so the Cancel button works inside WKWebView
        let closeScript = WKUserScript(
            source: """
            window.close = function() {
                window.webkit.messageHandlers.camelPadClose.postMessage({});
            };
            """,
            injectionTime: .atDocumentStart,
            forMainFrameOnly: true
        )
        config.userContentController.addUserScript(closeScript)
        config.userContentController.add(self, name: "camelPadClose")

        webView = WKWebView(
            frame: NSRect(x: 0, y: 0, width: width, height: height),
            configuration: config
        )
        webView.load(URLRequest(url: url))
    }

    required init?(coder: NSCoder) { fatalError() }
    override func loadView() { view = webView }

    func userContentController(_ controller: WKUserContentController, didReceive message: WKScriptMessage) {
        if message.name == "camelPadClose" { onClose?() }
    }
}

// ─── App delegate ─────────────────────────────────────────────────────────────

class AppDelegate: NSObject, NSApplicationDelegate {
    var statusItem: NSStatusItem!
    var popover: NSPopover?
    var popoverVC: PopoverWebViewController?
    var eventMonitor: Any?
    // Brief flag to prevent re-opening when the global event monitor and the
    // button action both fire for the same click on the tray icon.
    var justClosed = false

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        statusItem.button?.imageScaling = .scaleProportionallyDown
        statusItem.button?.action = #selector(statusBarButtonClicked(_:))
        statusItem.button?.target = self
        // Receive both left and right mouse-up so we can distinguish them
        statusItem.button?.sendAction(on: [.leftMouseUp, .rightMouseUp])

        emit(["type": "ready"])
        startStdinReader()
    }

    // ─── stdin reader ──────────────────────────────────────────────────────────

    func startStdinReader() {
        let thread = Thread {
            while let line = readLine(strippingNewline: true) {
                DispatchQueue.main.async { self.handleLine(line) }
            }
            DispatchQueue.main.async { NSApp.terminate(nil) }
        }
        thread.start()
    }

    func handleLine(_ line: String) {
        guard let data = line.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return }

        let type = obj["type"] as? String

        if type == nil {
            applyConfig(obj)
        } else if type == "update-item" {
            if let item = obj["item"] as? [String: Any] { handleUpdateItem(item) }
        } else if type == "show-popover" {
            let urlStr = obj["url"]    as? String  ?? ""
            let width  = obj["width"]  as? CGFloat ?? 520
            let height = obj["height"] as? CGFloat ?? 560
            if let url = URL(string: urlStr) { showPopover(url: url, width: width, height: height) }
        } else if type == "hide-popover" {
            closePopover()
        } else if type == "exit" {
            NSApp.terminate(nil)
        }
    }

    // ─── Config ────────────────────────────────────────────────────────────────

    func applyConfig(_ config: [String: Any]) {
        if let b64 = config["icon"] as? String, let data = Data(base64Encoded: b64) {
            let img = NSImage(data: data)
            img?.isTemplate = true
            statusItem.button?.image = img
        }
        if let tooltip = config["tooltip"] as? String {
            statusItem.button?.toolTip = tooltip
        }
    }

    // update-item: reflect connection status in the tooltip
    func handleUpdateItem(_ item: [String: Any]) {
        guard let title = item["title"] as? String else { return }
        let clean = title
            .replacingOccurrences(of: "● ", with: "")
            .replacingOccurrences(of: "○ ", with: "")
        statusItem.button?.toolTip = "Camel Pad · \(clean)"
    }

    // ─── Tray button click ─────────────────────────────────────────────────────

    @objc func statusBarButtonClicked(_ sender: NSStatusBarButton) {
        guard let event = NSApp.currentEvent else { return }

        if event.type == .rightMouseUp {
            showQuitMenu(sender)
            return
        }

        // Left click: toggle popover
        if let pop = popover, pop.isShown {
            closePopover()
        } else if !justClosed {
            // Ask TypeScript to start the settings server and call show-popover
            emit(["type": "tray-clicked"])
        }
    }

    // ─── Right-click Quit menu ─────────────────────────────────────────────────

    func showQuitMenu(_ button: NSStatusBarButton) {
        let menu = NSMenu()
        let item = NSMenuItem(
            title: "Quit Camel Pad",
            action: #selector(handleQuit(_:)),
            keyEquivalent: ""
        )
        item.target = self
        menu.addItem(item)
        // Assign to statusItem so the system positions the menu correctly
        // relative to the icon (the same way a normal status item menu works).
        statusItem.menu = menu
        statusItem.button?.performClick(nil)
        DispatchQueue.main.async { self.statusItem.menu = nil }
    }

    @objc func handleQuit(_ sender: NSMenuItem) {
        emit(["type": "quit-clicked"])
        // Give TypeScript a moment to shut down cleanly, then force-exit
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            NSApp.terminate(nil)
        }
    }

    // ─── Popover ──────────────────────────────────────────────────────────────

    func showPopover(url: URL, width: CGFloat, height: CGFloat) {
        if let existing = popover, existing.isShown { existing.close() }

        let vc = PopoverWebViewController(url: url, width: width, height: height) { [weak self] in
            self?.closePopover()
        }

        let pop = NSPopover()
        pop.contentSize = NSSize(width: width, height: height)
        pop.behavior = .applicationDefined  // we control dismissal
        pop.contentViewController = vc

        popover   = pop
        popoverVC = vc

        guard let button = statusItem.button else { return }
        pop.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
        pop.contentViewController?.view.window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)

        startEventMonitor()
    }

    func closePopover() {
        stopEventMonitor()
        popover?.close()
        popover    = nil
        popoverVC  = nil
        emit(["type": "popover-closed"])

        // Prevent the button action from immediately re-opening after a
        // global-monitor-triggered close (both fire for the same click).
        justClosed = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.15) { self.justClosed = false }
    }

    // ─── Global event monitor (dismiss popover on outside click) ───────────────

    func startEventMonitor() {
        stopEventMonitor()
        eventMonitor = NSEvent.addGlobalMonitorForEvents(
            matching: [.leftMouseDown, .rightMouseDown]
        ) { [weak self] _ in
            guard let self, let pop = self.popover, pop.isShown else { return }
            self.closePopover()
        }
    }

    func stopEventMonitor() {
        if let m = eventMonitor { NSEvent.removeMonitor(m); eventMonitor = nil }
    }
}

// ─── Entry point ──────────────────────────────────────────────────────────────

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
