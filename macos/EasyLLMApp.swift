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

final class EasyLLMAppDelegate: NSObject, NSApplicationDelegate, WKNavigationDelegate, WKUIDelegate {
    private var window: NSWindow?
    private var webView: WKWebView?
    private var serverProcess: Process?
    private var serverLogHandle: FileHandle?
    private var serverPort = 8022

    private var baseURL: URL {
        URL(string: "http://127.0.0.1:\(serverPort)")!
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        configureMenu()

        serverPort = firstAvailablePort(startingAt: 8022, limit: 80) ?? 8022
        createWindow()
        startServer()
        waitForServer()

        NSApp.activate(ignoringOtherApps: true)
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        true
    }

    func applicationWillTerminate(_ notification: Notification) {
        stopServer()
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
        NSApp.mainMenu = mainMenu
    }

    private func createWindow() {
        let configuration = WKWebViewConfiguration()
        configuration.preferences.javaScriptCanOpenWindowsAutomatically = true

        let webView = WKWebView(frame: .zero, configuration: configuration)
        webView.navigationDelegate = self
        webView.uiDelegate = self
        webView.underPageBackgroundColor = NSColor(red: 0.012, green: 0.027, blue: 0.071, alpha: 1)
        webView.wantsLayer = true
        webView.layer?.backgroundColor = NSColor(red: 0.012, green: 0.027, blue: 0.071, alpha: 1).cgColor
        webView.loadHTMLString(loadingHTML(), baseURL: nil)
        self.webView = webView

        let contentSize = NSSize(width: 1280, height: 820)
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
                width: contentSize.width,
                height: dragRegionHeight
            )
        )
        dragRegion.autoresizingMask = [.width, .minYMargin]
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
        window.minSize = NSSize(width: 960, height: 640)
        window.center()
        window.contentView = contentView
        window.makeKeyAndOrderFront(nil)
        self.window = window
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
            environment["DB_TYPE"] = "sqlite"
            environment["DB_SQLITE_PATH"] = dataURL.appendingPathComponent("easyllm.db").path
            environment["SECRET_KEY"] = try persistentSecret(in: supportURL)
            environment["EASYLLM_MAC_APP"] = "1"

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
                    self.webView?.load(URLRequest(url: self.baseURL))
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
}

private let app = NSApplication.shared
private let delegate = EasyLLMAppDelegate()
app.delegate = delegate
app.run()
