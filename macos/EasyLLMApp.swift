import AppKit
import Darwin
import Foundation
import WebKit

final class WindowDragRegionView: NSView {
    override var mouseDownCanMoveWindow: Bool {
        true
    }

    override func mouseDown(with event: NSEvent) {
        window?.performDrag(with: event)
    }
}

final class EasyLLMAppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate, WKNavigationDelegate, WKUIDelegate, WKScriptMessageHandler, NSMenuDelegate {
    private var window: NSWindow?
    private var webView: WKWebView?
    private var serverProcess: Process?
    private var serverLogHandle: FileHandle?
    private var serverPort = 8022
    private let launchCacheBuster = String(Int(Date().timeIntervalSince1970))
    private var statusItem: NSStatusItem?
    private var statusPollTimer: Timer?
    private var isRefreshingQuota = false
    private var lastSummary: AntigravitySummary?

    private var alwaysShowInMenuBar: Bool {
        get {
            if UserDefaults.standard.object(forKey: "alwaysShowInMenuBar") == nil {
                return true
            }
            return UserDefaults.standard.bool(forKey: "alwaysShowInMenuBar")
        }
        set {
            UserDefaults.standard.set(newValue, forKey: "alwaysShowInMenuBar")
            updateStatusItemVisibility()
        }
    }

    private var baseURL: URL {
        URL(string: "http://127.0.0.1:\(serverPort)")!
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        configureMenu()
        clearWebViewCache()

        setupStatusItem()
        updateStatusItemVisibility()
        startStatusPolling()

        serverPort = firstAvailablePort(startingAt: 8022, limit: 80) ?? 8022
        createWindow()
        startServer()
        waitForServer()

        NSApp.activate(ignoringOtherApps: true)
    }

    func windowShouldClose(_ sender: NSWindow) -> Bool {
        sender.orderOut(nil)
        webView?.isHidden = true
        updateStatusItemVisibility()
        notifyWebVisibility(false)
        return false
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        false
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        restoreWindow()
        return true
    }

    func applicationDidHide(_ notification: Notification) {
        webView?.isHidden = true
        updateStatusItemVisibility()
        notifyWebVisibility(false)
    }

    func applicationDidUnhide(_ notification: Notification) {
        if window?.isVisible == true && window?.isMiniaturized == false {
            webView?.isHidden = false
            notifyWebVisibility(true)
        }
        updateStatusItemVisibility()
    }

    func applicationWillTerminate(_ notification: Notification) {
        stopStatusPolling()
        if let statusItem {
            NSStatusBar.system.removeStatusItem(statusItem)
        }
        stopServer()
    }

    func windowDidMiniaturize(_ notification: Notification) {
        webView?.isHidden = true
        updateStatusItemVisibility()
        notifyWebVisibility(false)
        fetchAntigravitySummary()
    }

    func windowDidDeminiaturize(_ notification: Notification) {
        webView?.isHidden = false
        updateStatusItemVisibility()
        notifyWebVisibility(true)
    }

    func webView(
        _ webView: WKWebView,
        createWebViewWith configuration: WKWebViewConfiguration,
        for navigationAction: WKNavigationAction,
        windowFeatures: WKWindowFeatures
    ) -> WKWebView? {
        if navigationAction.targetFrame == nil, let url = navigationAction.request.url {
            NSWorkspace.shared.open(url)
        }
        return nil
    }

    func webView(
        _ webView: WKWebView,
        runOpenPanelWith parameters: WKOpenPanelParameters,
        initiatedByFrame frame: WKFrameInfo,
        completionHandler: @escaping ([URL]?) -> Void
    ) {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = parameters.allowsMultipleSelection
        panel.canChooseDirectories = parameters.allowsDirectories
        panel.canChooseFiles = !parameters.allowsDirectories
        panel.canCreateDirectories = false
        panel.resolvesAliases = true
        panel.title = parameters.allowsDirectories ? "选择要导入的文件夹" : "选择要导入的文件"
        panel.prompt = "选择"

        let finish: (NSApplication.ModalResponse) -> Void = { response in
            completionHandler(response == .OK ? panel.urls : nil)
        }

        if let window {
            panel.beginSheetModal(for: window, completionHandler: finish)
        } else {
            panel.begin(completionHandler: finish)
        }
    }

    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        guard message.name == "easyllmSaveFile",
              let payload = message.body as? [String: Any],
              let requestID = payload["request_id"] as? String,
              let filename = payload["filename"] as? String,
              let content = payload["content"] as? String
        else {
            return
        }
        saveFileFromWeb(requestID: requestID, filename: filename, content: content)
    }

    private func clearWebViewCache() {
        let dataTypes = WKWebsiteDataStore.allWebsiteDataTypes()
        WKWebsiteDataStore.default().removeData(
            ofTypes: dataTypes,
            modifiedSince: Date(timeIntervalSince1970: 0)
        ) {}
    }

    private func configureMenu() {
        let mainMenu = NSMenu()
        let appMenuItem = NSMenuItem()
        let appMenu = NSMenu()
        appMenu.addItem(
            NSMenuItem(
                title: "Quit EasyLLM",
                action: #selector(NSApplication.terminate(_:)),
                keyEquivalent: "q"
            )
        )
        appMenuItem.submenu = appMenu
        mainMenu.addItem(appMenuItem)

        let editMenuItem = NSMenuItem()
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(NSMenuItem(title: "Undo", action: #selector(UndoManager.undo), keyEquivalent: "z"))
        editMenu.addItem(NSMenuItem(title: "Redo", action: #selector(UndoManager.redo), keyEquivalent: "Z"))
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(NSMenuItem(title: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x"))
        editMenu.addItem(NSMenuItem(title: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c"))
        editMenu.addItem(NSMenuItem(title: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v"))
        editMenu.addItem(NSMenuItem(title: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a"))
        editMenuItem.submenu = editMenu
        mainMenu.addItem(editMenuItem)

        let windowMenuItem = NSMenuItem()
        let windowMenu = NSMenu(title: "Window")
        windowMenu.addItem(NSMenuItem(title: "Minimize", action: #selector(NSWindow.performMiniaturize(_:)), keyEquivalent: "m"))
        windowMenu.addItem(NSMenuItem(title: "Zoom", action: #selector(NSWindow.performZoom(_:)), keyEquivalent: ""))
        windowMenuItem.submenu = windowMenu
        mainMenu.addItem(windowMenuItem)

        NSApp.mainMenu = mainMenu
    }

    private func createWindow() {
        let configuration = WKWebViewConfiguration()
        configuration.preferences.javaScriptCanOpenWindowsAutomatically = true
        configuration.userContentController.add(self, name: "easyllmSaveFile")

        let webView = WKWebView(frame: .zero, configuration: configuration)
        webView.navigationDelegate = self
        webView.uiDelegate = self
        webView.underPageBackgroundColor = NSColor(red: 0.012, green: 0.027, blue: 0.071, alpha: 1)
        webView.wantsLayer = true
        webView.layer?.backgroundColor = NSColor(red: 0.012, green: 0.027, blue: 0.071, alpha: 1).cgColor
        webView.loadHTMLString(loadingHTML(), baseURL: nil)
        self.webView = webView

        let contentSize = NSSize(width: 1360, height: 820)
        let dragRegionHeight: CGFloat = 18
        let contentView = NSView(frame: NSRect(origin: .zero, size: contentSize))
        contentView.wantsLayer = true
        contentView.layer?.backgroundColor = NSColor(red: 0.012, green: 0.027, blue: 0.071, alpha: 1).cgColor

        webView.frame = contentView.bounds
        webView.autoresizingMask = [.width, .height]
        contentView.addSubview(webView)

        let dragRegion = WindowDragRegionView(
            frame: NSRect(
                x: 0,
                y: contentSize.height - dragRegionHeight,
                width: 240,
                height: dragRegionHeight
            )
        )
        dragRegion.autoresizingMask = [.minYMargin]
        contentView.addSubview(dragRegion)

        let window = NSWindow(
            contentRect: NSRect(origin: .zero, size: contentSize),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "EasyLLM"
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.styleMask.insert(.fullSizeContentView)
        window.isMovableByWindowBackground = true
        window.backgroundColor = NSColor(red: 0.012, green: 0.027, blue: 0.071, alpha: 1)
        window.minSize = NSSize(width: 800, height: 540)
        window.center()
        window.contentView = contentView
        window.delegate = self
        window.makeKeyAndOrderFront(nil)
        self.window = window
    }

    private func saveFileFromWeb(requestID: String, filename: String, content: String) {
        let panel = NSSavePanel()
        panel.nameFieldStringValue = filename
        panel.canCreateDirectories = true
        panel.title = "导出 EasyLLM 账号"
        panel.prompt = "保存"

        let finish: (NSApplication.ModalResponse) -> Void = { [weak self] response in
            guard let self else {
                return
            }
            guard response == .OK, let url = panel.url else {
                self.emitSaveFileResult(requestID: requestID, success: false, error: "cancelled")
                return
            }
            do {
                try content.write(to: url, atomically: true, encoding: .utf8)
                self.emitSaveFileResult(requestID: requestID, success: true, path: url.path)
            } catch {
                self.emitSaveFileResult(requestID: requestID, success: false, error: error.localizedDescription)
            }
        }

        if let window {
            panel.beginSheetModal(for: window, completionHandler: finish)
        } else {
            panel.begin(completionHandler: finish)
        }
    }

    private func emitSaveFileResult(requestID: String, success: Bool, path: String? = nil, error: String? = nil) {
        var payload: [String: Any] = [
            "request_id": requestID,
            "success": success,
        ]
        if let path {
            payload["path"] = path
        }
        if let error {
            payload["error"] = error
        }
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let json = String(data: data, encoding: .utf8)
        else {
            return
        }
        webView?.evaluateJavaScript(
            "window.dispatchEvent(new CustomEvent('easyllm-save-file-result', { detail: \(json) }))"
        )
    }

    private func startServer() {
        guard let executableURL = Bundle.main.url(forResource: "easyllm", withExtension: nil) else {
            showError("未找到内置 easyllm 后端二进制。请重新运行 scripts/build-macos-app.sh。")
            return
        }

        let resourcesURL = executableURL.deletingLastPathComponent()
        let supportURL = applicationSupportURL()
        let dataURL = supportURL.appendingPathComponent("data", isDirectory: true)
        let logURL = supportURL.appendingPathComponent("easyllm.log")

        do {
            try FileManager.default.createDirectory(at: dataURL, withIntermediateDirectories: true)
            if !FileManager.default.fileExists(atPath: logURL.path) {
                FileManager.default.createFile(atPath: logURL.path, contents: nil)
            }

            let logHandle = try FileHandle(forWritingTo: logURL)
            try logHandle.seekToEnd()
            serverLogHandle = logHandle

            var environment = ProcessInfo.processInfo.environment
            environment["SERVER_HOST"] = "127.0.0.1"
            environment["SERVER_PORT"] = "\(serverPort)"
            environment["DATA_DIR"] = dataURL.path
            environment["DB_SQLITE_PATH"] = dataURL.appendingPathComponent("easyllm.db").path
            environment["SECRET_KEY"] = try persistentSecret(in: supportURL)
            environment["EASYLLM_MAC_APP"] = "1"
            environment["GOMEMLIMIT"] = "48MiB"
            environment["GOGC"] = "50"

            let process = Process()
            process.executableURL = executableURL
            process.currentDirectoryURL = resourcesURL
            process.environment = environment
            process.standardOutput = logHandle
            process.standardError = logHandle
            try process.run()
            serverProcess = process
        } catch {
            showError("启动 EasyLLM 后端失败：\(error.localizedDescription)")
        }
    }

    private func stopServer() {
        guard let process = serverProcess else {
            return
        }

        if process.isRunning {
            process.terminate()
            let deadline = Date().addingTimeInterval(2)
            while process.isRunning && Date() < deadline {
                RunLoop.current.run(mode: .default, before: Date().addingTimeInterval(0.05))
            }
            if process.isRunning {
                kill(process.processIdentifier, SIGKILL)
            }
        }
        serverLogHandle?.closeFile()
    }

    private func waitForServer(attempt: Int = 0) {
        if attempt > 80 {
            showError("EasyLLM 后端启动超时。日志位置：\(applicationSupportURL().appendingPathComponent("easyllm.log").path)")
            return
        }

        var request = URLRequest(url: baseURL.appendingPathComponent("api/health"))
        request.timeoutInterval = 0.5

        URLSession.shared.dataTask(with: request) { [weak self] _, response, _ in
            guard let self else {
                return
            }

            let ok = (response as? HTTPURLResponse)?.statusCode == 200
            DispatchQueue.main.async {
                if ok {
                    var components = URLComponents(
                        url: self.baseURL.appendingPathComponent("codex"),
                        resolvingAgainstBaseURL: false
                    )
                    components?.queryItems = [
                        URLQueryItem(name: "mac_app", value: "1"),
                        URLQueryItem(name: "t", value: self.launchCacheBuster),
                    ]
                    let url = components?.url ?? self.baseURL.appendingPathComponent("codex")
                    var request = URLRequest(url: url)
                    request.cachePolicy = .reloadIgnoringLocalAndRemoteCacheData
                    self.webView?.load(request)
                    self.fetchAntigravitySummary()
                } else {
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.25) {
                        self.waitForServer(attempt: attempt + 1)
                    }
                }
            }
        }.resume()
    }

    private func applicationSupportURL() -> URL {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        let supportURL = base.appendingPathComponent("EasyLLM", isDirectory: true)
        try? FileManager.default.createDirectory(at: supportURL, withIntermediateDirectories: true)
        return supportURL
    }

    private func persistentSecret(in supportURL: URL) throws -> String {
        let secretURL = supportURL.appendingPathComponent("secret.key")
        if let existing = try? String(contentsOf: secretURL, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines),
           !existing.isEmpty {
            return existing
        }

        let secret = (0..<4)
            .map { _ in UUID().uuidString.replacingOccurrences(of: "-", with: "") }
            .joined()
        try secret.write(to: secretURL, atomically: true, encoding: .utf8)
        chmod(secretURL.path, S_IRUSR | S_IWUSR)
        return secret
    }

    private func firstAvailablePort(startingAt start: Int, limit: Int) -> Int? {
        for port in start..<(start + limit) {
            if isPortAvailable(port) {
                return port
            }
        }
        return nil
    }

    private func isPortAvailable(_ port: Int) -> Bool {
        let socketFD = socket(AF_INET, SOCK_STREAM, 0)
        if socketFD < 0 {
            return false
        }
        defer {
            close(socketFD)
        }

        var reuse: Int32 = 1
        setsockopt(socketFD, SOL_SOCKET, SO_REUSEADDR, &reuse, socklen_t(MemoryLayout<Int32>.size))

        var address = sockaddr_in()
        address.sin_len = UInt8(MemoryLayout<sockaddr_in>.size)
        address.sin_family = sa_family_t(AF_INET)
        address.sin_port = in_port_t(port).bigEndian
        address.sin_addr = in_addr(s_addr: inet_addr("127.0.0.1"))

        let bindResult = withUnsafePointer(to: &address) { pointer in
            pointer.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockaddrPointer in
                Darwin.bind(socketFD, sockaddrPointer, socklen_t(MemoryLayout<sockaddr_in>.size))
            }
        }

        return bindResult == 0
    }

    private func loadingHTML() -> String {
        """
        <!doctype html>
        <html lang="zh-CN">
        <head>
          <meta charset="utf-8">
          <style>
            body {
              margin: 0;
              height: 100vh;
              display: grid;
              place-items: center;
              background: #0f172a;
              color: #dbeafe;
              font: 14px -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
            }
            .box {
              width: min(420px, calc(100vw - 48px));
              padding: 24px;
              border: 1px solid rgba(148, 163, 184, 0.24);
              border-radius: 10px;
              background: rgba(15, 23, 42, 0.72);
            }
            h1 { margin: 0 0 8px; font-size: 20px; }
            p { margin: 0; color: #94a3b8; line-height: 1.6; }
          </style>
        </head>
        <body>
          <div class="box">
            <h1>正在启动 EasyLLM</h1>
            <p>本地后端服务启动后会自动进入管理界面。</p>
          </div>
        </body>
        </html>
        """
    }

    private func showError(_ message: String) {
        webView?.loadHTMLString(
            """
            <!doctype html>
            <meta charset="utf-8">
            <body style="margin:0;padding:32px;background:#111827;color:#fecaca;font:14px -apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
              <h1 style="font-size:20px;color:#fff;">EasyLLM 启动失败</h1>
              <pre style="white-space:pre-wrap;line-height:1.6;">\(escapeHTML(message))</pre>
            </body>
            """,
            baseURL: nil
        )
    }

    private func escapeHTML(_ value: String) -> String {
        value
            .replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
            .replacingOccurrences(of: "\"", with: "&quot;")
    }

    // MARK: - Status Bar & Antigravity Quota

    struct AntigravitySummary: Codable {
        let hasAccount: Bool
        let accountId: String?
        let email: String?
        let displayName: String?
        let subscriptionTier: String?
        let status: String?
        let claude5h: Int?
        let claude5hResetFormatted: String?
        let claudeWeekly: Int?
        let claudeWeeklyResetFormatted: String?
        let gemini5h: Int?
        let gemini5hResetFormatted: String?
        let geminiWeekly: Int?
        let geminiWeeklyResetFormatted: String?
        let statusText: String?
        let tooltipText: String?
        let lastRefreshAt: String?

        enum CodingKeys: String, CodingKey {
            case hasAccount = "has_account"
            case accountId = "account_id"
            case email
            case displayName = "display_name"
            case subscriptionTier = "subscription_tier"
            case status
            case claude5h = "claude_5h"
            case claude5hResetFormatted = "claude_5h_reset_formatted"
            case claudeWeekly = "claude_weekly"
            case claudeWeeklyResetFormatted = "claude_weekly_reset_formatted"
            case gemini5h = "gemini_5h"
            case gemini5hResetFormatted = "gemini_5h_reset_formatted"
            case geminiWeekly = "gemini_weekly"
            case geminiWeeklyResetFormatted = "gemini_weekly_reset_formatted"
            case statusText = "status_text"
            case tooltipText = "tooltip_text"
            case lastRefreshAt = "last_refresh_at"
        }
    }

    private func updateStatusItemVisibility() {
        let isWindowHidden = (window == nil) || (window?.isVisible == false) || (window?.isMiniaturized == true) || NSApp.isHidden
        let shouldShow = alwaysShowInMenuBar || isWindowHidden
        if shouldShow {
            if statusItem == nil {
                setupStatusItem()
            }
            statusItem?.isVisible = true
            updateStatusItemDisplay()
        } else {
            statusItem?.isVisible = false
        }
    }

    private func setupStatusItem() {
        guard statusItem == nil else { return }
        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem = item
        updateStatusItemDisplay()
    }

    private func updateStatusItemDisplay() {
        guard let statusItem else { return }

        let title: String
        let toolTip: String
        let isLowQuota: Bool
        if let summary = lastSummary {
            title = summary.statusText ?? "✦ EasyLLM"
            toolTip = summary.tooltipText ?? "EasyLLM"
            let cLow = (summary.claude5h ?? 100) <= 20
            let gLow = (summary.gemini5h ?? 100) <= 20
            isLowQuota = cLow || gLow
        } else {
            title = "✦ EasyLLM"
            toolTip = "EasyLLM - 正在连接后端服务..."
            isLowQuota = false
        }

        if let button = statusItem.button {
            let symbolName = isLowQuota ? "exclamationmark.triangle.fill" : "sparkles"
            if let img = NSImage(systemSymbolName: symbolName, accessibilityDescription: "EasyLLM") {
                img.isTemplate = true
                button.image = img
                button.imagePosition = .imageLeft
            }
            let cleanTitle = title.replacingOccurrences(of: "✦ ", with: "").replacingOccurrences(of: "⚠️ ", with: "")
            button.title = " " + cleanTitle
            button.toolTip = toolTip
        }

        statusItem.menu = buildStatusBarMenu()
    }

    private func buildStatusBarMenu() -> NSMenu {
        let menu = NSMenu()
        menu.delegate = self

        let openItem = NSMenuItem(title: "打开 EasyLLM 窗口", action: #selector(restoreWindow), keyEquivalent: "o")
        openItem.target = self
        openItem.attributedTitle = NSAttributedString(
            string: "打开 EasyLLM 窗口",
            attributes: [.font: NSFont.boldSystemFont(ofSize: 13)]
        )
        menu.addItem(openItem)
        menu.addItem(NSMenuItem.separator())

        if let summary = lastSummary, summary.hasAccount {
            let name = summary.displayName ?? summary.email ?? "未命名账号"
            let tier = summary.subscriptionTier.map { " [\($0)]" } ?? ""
            let accHeader = NSMenuItem(title: "👤 \(name)\(tier)", action: nil, keyEquivalent: "")
            accHeader.isEnabled = false
            menu.addItem(accHeader)

            menu.addItem(NSMenuItem.separator())

            let c5 = summary.claude5h.map { "\($0)%" } ?? "--"
            let c5Reset: String
            if let rf = summary.claude5hResetFormatted, !rf.isEmpty {
                c5Reset = " (\(rf))"
            } else {
                c5Reset = ""
            }
            let c5Item = NSMenuItem(title: "Claude (5h): \(c5)\(c5Reset)", action: nil, keyEquivalent: "")
            c5Item.isEnabled = false
            menu.addItem(c5Item)

            let cW = summary.claudeWeekly.map { "\($0)%" } ?? "--"
            let cWItem = NSMenuItem(title: "Claude (周): \(cW)", action: nil, keyEquivalent: "")
            cWItem.isEnabled = false
            menu.addItem(cWItem)

            let g5 = summary.gemini5h.map { "\($0)%" } ?? "--"
            let g5Reset: String
            if let rf = summary.gemini5hResetFormatted, !rf.isEmpty {
                g5Reset = " (\(rf))"
            } else {
                g5Reset = ""
            }
            let g5Item = NSMenuItem(title: "Gemini (5h): \(g5)\(g5Reset)", action: nil, keyEquivalent: "")
            g5Item.isEnabled = false
            menu.addItem(g5Item)

            let gW = summary.geminiWeekly.map { "\($0)%" } ?? "--"
            let gWItem = NSMenuItem(title: "Gemini (周): \(gW)", action: nil, keyEquivalent: "")
            gWItem.isEnabled = false
            menu.addItem(gWItem)

            if let lastRef = summary.lastRefreshAt, !lastRef.isEmpty {
                menu.addItem(NSMenuItem.separator())
                let timeItem = NSMenuItem(title: "🕒 刷新时间: \(lastRef)", action: nil, keyEquivalent: "")
                timeItem.isEnabled = false
                menu.addItem(timeItem)
            }
        } else {
            let emptyItem = NSMenuItem(title: "暂无 Antigravity 账号数据", action: nil, keyEquivalent: "")
            emptyItem.isEnabled = false
            menu.addItem(emptyItem)
        }

        menu.addItem(NSMenuItem.separator())

        let refreshTitle = isRefreshingQuota ? "正在刷新配额..." : "立即刷新配额"
        let refreshItem = NSMenuItem(title: refreshTitle, action: #selector(refreshQuotaAction), keyEquivalent: "r")
        refreshItem.target = self
        refreshItem.isEnabled = !isRefreshingQuota
        menu.addItem(refreshItem)

        let toggleTitle = alwaysShowInMenuBar ? "✓ 始终在状态栏常驻" : "仅最小化/隐藏时显示"
        let toggleItem = NSMenuItem(title: toggleTitle, action: #selector(toggleAlwaysShow), keyEquivalent: "")
        toggleItem.target = self
        toggleItem.state = alwaysShowInMenuBar ? .on : .off
        menu.addItem(toggleItem)

        menu.addItem(NSMenuItem.separator())

        let quitItem = NSMenuItem(title: "退出 EasyLLM", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        menu.addItem(quitItem)

        return menu
    }

    private func fetchAntigravitySummary() {
        let url = baseURL.appendingPathComponent("api/v1/antigravity/summary")
        var request = URLRequest(url: url)
        request.timeoutInterval = 5
        request.cachePolicy = .reloadIgnoringLocalAndRemoteCacheData

        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            guard let self, let data,
                  let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200,
                  let summary = try? JSONDecoder().decode(AntigravitySummary.self, from: data) else {
                return
            }
            DispatchQueue.main.async {
                self.lastSummary = summary
                self.updateStatusItemDisplay()
            }
        }.resume()
    }

    @objc private func refreshQuotaAction() {
        guard !isRefreshingQuota else { return }
        isRefreshingQuota = true
        updateStatusItemDisplay()

        let url = baseURL.appendingPathComponent("api/v1/antigravity/summary/refresh")
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.timeoutInterval = 25
        request.cachePolicy = .reloadIgnoringLocalAndRemoteCacheData

        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            guard let self else { return }
            DispatchQueue.main.async {
                self.isRefreshingQuota = false
                if let data,
                   let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200,
                   let summary = try? JSONDecoder().decode(AntigravitySummary.self, from: data) {
                    self.lastSummary = summary
                }
                self.updateStatusItemDisplay()
            }
        }.resume()
    }

    @objc private func restoreWindow() {
        guard let window else { return }
        webView?.isHidden = false
        if window.isMiniaturized {
            window.deminiaturize(nil)
        }
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        notifyWebVisibility(true)
    }

    private func notifyWebVisibility(_ visible: Bool) {
        let script = "window.__easyllm_hidden = \(!visible); window.dispatchEvent(new CustomEvent('easyllm-app-visibility', { detail: { visible: \(visible) } }));"
        webView?.evaluateJavaScript(script, completionHandler: nil)
    }

    func menuWillOpen(_ menu: NSMenu) {
        fetchAntigravitySummary()
    }

    @objc private func toggleAlwaysShow() {
        alwaysShowInMenuBar = !alwaysShowInMenuBar
    }

    private func startStatusPolling() {
        stopStatusPolling()
        // 120s interval with 15s coalescing tolerance so macOS can idle and save CPU & battery
        let timer = Timer(timeInterval: 120.0, repeats: true) { [weak self] _ in
            guard let self else { return }
            if self.statusItem?.isVisible == true {
                self.fetchAntigravitySummary()
            }
        }
        timer.tolerance = 15.0
        RunLoop.main.add(timer, forMode: .common)
        statusPollTimer = timer
    }

    private func stopStatusPolling() {
        statusPollTimer?.invalidate()
        statusPollTimer = nil
    }
}

private let app = NSApplication.shared
private let delegate = EasyLLMAppDelegate()
app.delegate = delegate
app.run()
