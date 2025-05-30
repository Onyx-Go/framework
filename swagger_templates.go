package onyx

// HTML templates for Swagger UI and developer tools

const swaggerUIIndexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{.Title}} - Onyx</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@4.15.5/favicon-32x32.png" sizes="32x32" />
    <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@4.15.5/favicon-16x16.png" sizes="16x16" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
        
        .custom-topbar {
            background: #1f1f1f;
            padding: 10px 20px;
            color: white;
            border-bottom: 1px solid #333;
        }
        
        .custom-topbar h1 {
            margin: 0;
            font-size: 1.5em;
            display: inline-block;
        }
        
        .custom-nav {
            float: right;
            margin-top: 5px;
        }
        
        .custom-nav a {
            color: #61affe;
            text-decoration: none;
            margin-left: 20px;
            padding: 5px 10px;
            border-radius: 3px;
            transition: background 0.3s;
        }
        
        .custom-nav a:hover {
            background: #333;
        }
        
        .version-selector {
            display: inline-block;
            margin-left: 20px;
        }
        
        .version-selector select {
            background: #333;
            color: white;
            border: 1px solid #555;
            padding: 5px 10px;
            border-radius: 3px;
        }
        
        .dark-mode-toggle {
            background: none;
            border: 1px solid #61affe;
            color: #61affe;
            padding: 5px 10px;
            border-radius: 3px;
            cursor: pointer;
            margin-left: 10px;
        }
        
        .feature-notice {
            background: #e3f2fd;
            border-left: 4px solid #2196f3;
            padding: 10px 20px;
            margin: 20px;
            border-radius: 3px;
        }
        
        .deprecated-warning {
            background: #fff3cd;
            border-left: 4px solid #ffc107;
            padding: 10px 20px;
            margin: 20px;
            border-radius: 3px;
            color: #856404;
        }
        
        .eol-warning {
            background: #f8d7da;
            border-left: 4px solid #dc3545;
            padding: 10px 20px;
            margin: 20px;
            border-radius: 3px;
            color: #721c24;
        }
        
        {{if .Config.CustomCSS}}{{.Config.CustomCSS}}{{end}}
    </style>
</head>
<body>
    <div class="custom-topbar">
        <h1>{{.Title}}</h1>
        <div class="custom-nav">
            {{if .Features.Versioning}}
            <a href="{{.BasePath}}/versions">📋 Versions</a>
            {{end}}
            {{if .Features.Playground}}
            <a href="{{.BasePath}}/playground">🎮 Playground</a>
            {{end}}
            {{if .Features.CodeGenerator}}
            <a href="{{.BasePath}}/code-gen">⚡ Code Gen</a>
            {{end}}
            <a href="{{.BasePath}}/health">❤️ Health</a>
            <button class="dark-mode-toggle" onclick="toggleDarkMode()">🌙 Dark</button>
            
            {{if .AvailableVersions}}
            <div class="version-selector">
                <select onchange="switchVersion(this.value)">
                    {{range .AvailableVersions}}
                    <option value="{{.}}" {{if eq . $.CurrentVersion}}selected{{end}}>{{.}}</option>
                    {{end}}
                </select>
            </div>
            {{end}}
        </div>
        <div style="clear: both;"></div>
    </div>
    
    {{if .VersionInfo}}
    {{if .VersionInfo.Deprecated}}
    <div class="deprecated-warning">
        <strong>⚠️ This API version is deprecated</strong>
        {{if .VersionInfo.EOLDate}} and will be discontinued on {{.VersionInfo.EOLDate.Format "2006-01-02"}}{{end}}.
        Please consider migrating to a newer version.
    </div>
    {{end}}
    {{if eq .VersionInfo.Status "eol"}}
    <div class="eol-warning">
        <strong>🚫 This API version is no longer supported</strong>
        This version has reached end-of-life and should not be used in production.
    </div>
    {{end}}
    {{end}}
    
    {{if .Features}}
    <div class="feature-notice">
        <strong>🚀 Enhanced Documentation Features Available:</strong>
        {{if .Features.Versioning}}API Versioning{{end}}
        {{if .Features.Playground}}• Interactive Playground{{end}}
        {{if .Features.CodeGenerator}}• Code Generation{{end}}
    </div>
    {{end}}
    
    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const config = {
                url: '{{.Config.SpecURL}}',
                dom_id: '#swagger-ui',
                deepLinking: {{.Config.DeepLinking}},
                displayOperationId: {{.Config.DisplayOperationId}},
                defaultModelsExpandDepth: {{.Config.DefaultModelsExpandDepth}},
                defaultModelExpandDepth: {{.Config.DefaultModelExpandDepth}},
                defaultModelRendering: '{{.Config.DefaultModelRendering}}',
                displayRequestDuration: {{.Config.DisplayRequestDuration}},
                docExpansion: '{{.Config.DocExpansion}}',
                filter: {{.Config.Filter}},
                maxDisplayedTags: {{.Config.MaxDisplayedTags}},
                showExtensions: {{.Config.ShowExtensions}},
                showCommonExtensions: {{.Config.ShowCommonExtensions}},
                tryItOutEnabled: {{.Config.TryItOutEnabled}},
                supportedSubmitMethods: {{.Config.SupportedSubmitMethods | json}},
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                onComplete: function() {
                    console.log('Swagger UI loaded successfully');
                    
                    // Add custom functionality
                    addExportButtons();
                    addVersionInfo();
                },
                onFailure: function(error) {
                    console.error('Failed to load Swagger UI:', error);
                }
            };
            
            {{if .Config.OAuth}}
            config.oauth = {{.Config.OAuth | json}};
            {{end}}
            
            {{if .Config.RequestInterceptor}}
            config.requestInterceptor = {{.Config.RequestInterceptor}};
            {{end}}
            
            {{if .Config.ResponseInterceptor}}
            config.responseInterceptor = {{.Config.ResponseInterceptor}};
            {{end}}
            
            const ui = SwaggerUIBundle(config);
            window.swaggerUI = ui;
        };
        
        function switchVersion(version) {
            if (version) {
                window.location.href = '{{.BasePath}}/version/' + version;
            }
        }
        
        function addExportButtons() {
            const toolbar = document.querySelector('.download-url-wrapper');
            if (toolbar) {
                const exportBtn = document.createElement('button');
                exportBtn.innerHTML = '📥 Export';
                exportBtn.className = 'btn';
                exportBtn.onclick = function() {
                    const menu = document.createElement('div');
                    menu.innerHTML = 
                        '<div style="position: absolute; background: white; border: 1px solid #ccc; box-shadow: 0 2px 10px rgba(0,0,0,0.1); z-index: 1000; padding: 10px;">' +
                            '<a href="{{.Config.SpecURL}}" download="openapi.json" style="display: block; padding: 5px 10px; text-decoration: none;">JSON</a>' +
                            '<a href="{{.Config.SpecURL}}?format=yaml" download="openapi.yaml" style="display: block; padding: 5px 10px; text-decoration: none;">YAML</a>' +
                            '<a href="#" onclick="exportPostman()" style="display: block; padding: 5px 10px; text-decoration: none;">Postman Collection</a>' +
                        '</div>';
                    document.body.appendChild(menu);
                    
                    setTimeout(() => {
                        document.body.removeChild(menu);
                    }, 5000);
                };
                toolbar.appendChild(exportBtn);
            }
        }
        
        function addVersionInfo() {
            {{if .Version}}
            const info = document.querySelector('.info');
            if (info) {
                const versionBadge = document.createElement('div');
                versionBadge.innerHTML = '<span style="background: #61affe; color: white; padding: 2px 8px; border-radius: 3px; font-size: 0.8em;">v{{.Version}}</span>';
                info.appendChild(versionBadge);
            }
            {{end}}
        }
        
        function exportPostman() {
            fetch('{{.Config.SpecURL}}')
                .then(response => response.json())
                .then(spec => {
                    // Convert OpenAPI to Postman collection
                    console.log('Converting to Postman collection...');
                    // This would need a proper OpenAPI to Postman converter
                });
        }
        
        // Dark mode functionality
        function toggleDarkMode() {
            document.body.classList.toggle('dark-mode');
            localStorage.setItem('darkMode', document.body.classList.contains('dark-mode'));
        }
        
        // Load dark mode preference
        if (localStorage.getItem('darkMode') === 'true') {
            document.body.classList.add('dark-mode');
        }
        
        {{if .Config.CustomJS}}
        {{.Config.CustomJS}}
        {{end}}
    </script>
</body>
</html>`

const versionsTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{.Title}} - Onyx</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; background: #fafafa; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: #1f1f1f; color: white; padding: 20px; margin: -20px -20px 20px -20px; }
        .header h1 { margin: 0; }
        .header nav { margin-top: 10px; }
        .header a { color: #61affe; text-decoration: none; margin-right: 20px; }
        .header a:hover { text-decoration: underline; }
        
        .version-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .version-card { background: white; border: 1px solid #ddd; border-radius: 8px; padding: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .version-card h3 { margin: 0 0 10px 0; color: #333; }
        .version-card .version-number { font-size: 1.2em; font-weight: bold; color: #61affe; }
        .version-card .status { padding: 2px 8px; border-radius: 4px; font-size: 0.8em; font-weight: bold; }
        .status.stable { background: #d4edda; color: #155724; }
        .status.deprecated { background: #fff3cd; color: #856404; }
        .status.eol { background: #f8d7da; color: #721c24; }
        .status.development { background: #cce5ff; color: #004085; }
        
        .version-meta { margin: 10px 0; font-size: 0.9em; color: #666; }
        .version-actions { margin-top: 15px; }
        .btn { display: inline-block; padding: 8px 16px; background: #61affe; color: white; text-decoration: none; border-radius: 4px; margin-right: 10px; }
        .btn:hover { background: #4e90d9; }
        .btn.secondary { background: #6c757d; }
        .btn.secondary:hover { background: #545b62; }
        
        .comparison-section { margin-top: 40px; padding: 20px; background: white; border-radius: 8px; }
        .compare-form { display: flex; gap: 10px; align-items: center; }
        .compare-form select { padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
            <nav>
                <a href="{{.BasePath}}">📖 Documentation</a>
                <a href="{{.BasePath}}/playground">🎮 Playground</a>
                <a href="{{.BasePath}}/code-gen">⚡ Code Generator</a>
                <a href="/docs/api/changelog">📋 Changelog</a>
            </nav>
        </div>
        
        <h2>Available API Versions</h2>
        <div class="version-grid">
            {{range .Versions}}
            <div class="version-card">
                <h3>
                    <span class="version-number">{{.Version}}</span>
                    {{if .Name}}- {{.Name}}{{end}}
                </h3>
                <div class="status {{.Status}}">{{.Status}}</div>
                
                <div class="version-meta">
                    <div><strong>Released:</strong> {{.Released.Format "2006-01-02"}}</div>
                    {{if .Deprecated}}
                    <div><strong>⚠️ Deprecated:</strong> {{.DeprecatedAt.Format "2006-01-02"}}</div>
                    {{end}}
                    {{if .EOLDate}}
                    <div><strong>🚫 End of Life:</strong> {{.EOLDate.Format "2006-01-02"}}</div>
                    {{end}}
                </div>
                
                {{if .Description}}
                <p>{{.Description}}</p>
                {{end}}
                
                <div class="version-actions">
                    <a href="{{$.BasePath}}/version/{{.Version}}" class="btn">📖 View Docs</a>
                    <a href="/docs/{{.Version}}/openapi.json" class="btn secondary">📥 Download Spec</a>
                </div>
            </div>
            {{end}}
        </div>
        
        <div class="comparison-section">
            <h3>Compare Versions</h3>
            <div class="compare-form">
                <select id="version1">
                    <option value="">Select version 1</option>
                    {{range .Versions}}
                    <option value="{{.Version}}">{{.Version}}{{if .Name}} - {{.Name}}{{end}}</option>
                    {{end}}
                </select>
                <span>vs</span>
                <select id="version2">
                    <option value="">Select version 2</option>
                    {{range .Versions}}
                    <option value="{{.Version}}">{{.Version}}{{if .Name}} - {{.Name}}{{end}}</option>
                    {{end}}
                </select>
                <button class="btn" onclick="compareVersions()">Compare</button>
            </div>
            <div id="comparison-result" style="margin-top: 20px;"></div>
        </div>
    </div>
    
    <script>
        function compareVersions() {
            const version1 = document.getElementById('version1').value;
            const version2 = document.getElementById('version2').value;
            
            if (!version1 || !version2) {
                alert('Please select both versions to compare');
                return;
            }
            
            fetch('/docs/api/compare/' + version1 + '/' + version2)
                .then(response => response.json())
                .then(data => {
                    displayComparison(data);
                })
                .catch(error => {
                    console.error('Comparison failed:', error);
                    document.getElementById('comparison-result').innerHTML = 
                        '<div style="color: red;">Comparison failed: ' + error + '</div>';
                });
        }
        
        function displayComparison(data) {
            const result = document.getElementById('comparison-result');
            let html = '<h4>Comparison Results</h4>';
            
            if (data.added && data.added.length > 0) {
                html += '<h5>➕ Added in ' + data.version2 + ':</h5><ul>';
                data.added.forEach(item => html += '<li>' + item + '</li>');
                html += '</ul>';
            }
            
            if (data.removed && data.removed.length > 0) {
                html += '<h5>➖ Removed in ' + data.version2 + ':</h5><ul>';
                data.removed.forEach(item => html += '<li>' + item + '</li>');
                html += '</ul>';
            }
            
            if (data.modified && data.modified.length > 0) {
                html += '<h5>🔄 Modified in ' + data.version2 + ':</h5><ul>';
                data.modified.forEach(item => html += '<li>' + item + '</li>');
                html += '</ul>';
            }
            
            if (!data.added.length && !data.removed.length && !data.modified.length) {
                html += '<p>No differences found between these versions.</p>';
            }
            
            result.innerHTML = html;
        }
    </script>
</body>
</html>`

const playgroundTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{.Title}} - API Playground</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { background: #1f1f1f; color: white; padding: 20px; }
        .header h1 { margin: 0; }
        .header nav { margin-top: 10px; }
        .header a { color: #61affe; text-decoration: none; margin-right: 20px; }
        
        .playground { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; padding: 20px; height: calc(100vh - 120px); }
        .panel { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .panel-header { background: #f8f9fa; padding: 15px; border-bottom: 1px solid #dee2e6; font-weight: bold; }
        .panel-content { padding: 20px; height: calc(100% - 70px); overflow: auto; }
        
        .form-group { margin-bottom: 15px; }
        .form-group label { display: block; margin-bottom: 5px; font-weight: bold; }
        .form-control { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; }
        .btn { padding: 10px 20px; background: #61affe; color: white; border: none; border-radius: 4px; cursor: pointer; }
        .btn:hover { background: #4e90d9; }
        
        .method-badge { padding: 2px 8px; border-radius: 4px; font-size: 0.8em; font-weight: bold; margin-right: 10px; }
        .method-get { background: #61affe; color: white; }
        .method-post { background: #49cc90; color: white; }
        .method-put { background: #fca130; color: white; }
        .method-delete { background: #f93e3e; color: white; }
        
        .response-section { margin-top: 20px; }
        .response-header { background: #f8f9fa; padding: 10px; border-radius: 4px 4px 0 0; border: 1px solid #dee2e6; }
        .response-body { background: #f8f9fa; padding: 15px; border: 1px solid #dee2e6; border-top: none; border-radius: 0 0 4px 4px; }
        .response-body pre { margin: 0; white-space: pre-wrap; word-break: break-all; }
        
        .endpoint-list { max-height: 300px; overflow-y: auto; border: 1px solid #ddd; border-radius: 4px; }
        .endpoint-item { padding: 10px; border-bottom: 1px solid #eee; cursor: pointer; }
        .endpoint-item:hover { background: #f8f9fa; }
        .endpoint-item:last-child { border-bottom: none; }
        
        .tabs { border-bottom: 1px solid #dee2e6; margin-bottom: 15px; }
        .tab { display: inline-block; padding: 10px 15px; cursor: pointer; border-bottom: 2px solid transparent; }
        .tab.active { border-bottom-color: #61affe; color: #61affe; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
            <nav>
                <a href="{{.BasePath}}">📖 Documentation</a>
                <a href="{{.BasePath}}/versions">📋 Versions</a>
                <a href="{{.BasePath}}/code-gen">⚡ Code Generator</a>
            </nav>
        </div>
        
        <div class="playground">
            <div class="panel">
                <div class="panel-header">🎮 API Request Builder</div>
                <div class="panel-content">
                    <div class="form-group">
                        <label>Available Endpoints</label>
                        <div class="endpoint-list" id="endpoints">
                            <div class="endpoint-item" onclick="selectEndpoint('GET', '/api/v1/users')">
                                <span class="method-badge method-get">GET</span>
                                /api/v1/users
                            </div>
                            <div class="endpoint-item" onclick="selectEndpoint('POST', '/api/v1/users')">
                                <span class="method-badge method-post">POST</span>
                                /api/v1/users
                            </div>
                            <div class="endpoint-item" onclick="selectEndpoint('GET', '/api/v1/users/{id}')">
                                <span class="method-badge method-get">GET</span>
                                /api/v1/users/{id}
                            </div>
                            <div class="endpoint-item" onclick="selectEndpoint('PUT', '/api/v1/users/{id}')">
                                <span class="method-badge method-put">PUT</span>
                                /api/v1/users/{id}
                            </div>
                            <div class="endpoint-item" onclick="selectEndpoint('DELETE', '/api/v1/users/{id}')">
                                <span class="method-badge method-delete">DELETE</span>
                                /api/v1/users/{id}
                            </div>
                        </div>
                    </div>
                    
                    <div class="form-group">
                        <label>Method</label>
                        <select class="form-control" id="method">
                            <option value="GET">GET</option>
                            <option value="POST">POST</option>
                            <option value="PUT">PUT</option>
                            <option value="DELETE">DELETE</option>
                            <option value="PATCH">PATCH</option>
                        </select>
                    </div>
                    
                    <div class="form-group">
                        <label>URL</label>
                        <input type="text" class="form-control" id="url" placeholder="https://api.example.com/v1/users">
                    </div>
                    
                    <div class="tabs">
                        <div class="tab active" onclick="switchTab('headers')">Headers</div>
                        <div class="tab" onclick="switchTab('params')">Parameters</div>
                        <div class="tab" onclick="switchTab('body')">Body</div>
                        <div class="tab" onclick="switchTab('auth')">Auth</div>
                    </div>
                    
                    <div id="headers-tab" class="tab-content active">
                        <div class="form-group">
                            <label>Headers (JSON format)</label>
                            <textarea class="form-control" id="headers" rows="4" placeholder='{"Content-Type": "application/json", "Authorization": "Bearer token"}'></textarea>
                        </div>
                    </div>
                    
                    <div id="params-tab" class="tab-content">
                        <div class="form-group">
                            <label>Query Parameters (JSON format)</label>
                            <textarea class="form-control" id="params" rows="4" placeholder='{"page": 1, "limit": 10}'></textarea>
                        </div>
                    </div>
                    
                    <div id="body-tab" class="tab-content">
                        <div class="form-group">
                            <label>Request Body (JSON format)</label>
                            <textarea class="form-control" id="body" rows="8" placeholder='{"name": "John Doe", "email": "john@example.com"}'></textarea>
                        </div>
                    </div>
                    
                    <div id="auth-tab" class="tab-content">
                        <div class="form-group">
                            <label>Authentication Type</label>
                            <select class="form-control" id="authType">
                                <option value="">None</option>
                                <option value="bearer">Bearer Token</option>
                                <option value="basic">Basic Auth</option>
                                <option value="apikey">API Key</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Token/Key</label>
                            <input type="text" class="form-control" id="authToken" placeholder="Enter token or key">
                        </div>
                    </div>
                    
                    <button class="btn" onclick="sendRequest()">🚀 Send Request</button>
                </div>
            </div>
            
            <div class="panel">
                <div class="panel-header">📊 Response</div>
                <div class="panel-content">
                    <div id="response-container">
                        <p style="color: #666; text-align: center; margin-top: 50px;">
                            Send a request to see the response here
                        </p>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        function switchTab(tabName) {
            // Hide all tab contents
            document.querySelectorAll('.tab-content').forEach(tab => tab.classList.remove('active'));
            document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
            
            // Show selected tab
            document.getElementById(tabName + '-tab').classList.add('active');
            event.target.classList.add('active');
        }
        
        function selectEndpoint(method, path) {
            document.getElementById('method').value = method;
            document.getElementById('url').value = window.location.origin + path;
        }
        
        function sendRequest() {
            const method = document.getElementById('method').value;
            const url = document.getElementById('url').value;
            const headersText = document.getElementById('headers').value;
            const paramsText = document.getElementById('params').value;
            const bodyText = document.getElementById('body').value;
            const authType = document.getElementById('authType').value;
            const authToken = document.getElementById('authToken').value;
            
            // Parse headers
            let headers = {};
            if (headersText) {
                try {
                    headers = JSON.parse(headersText);
                } catch (e) {
                    alert('Invalid headers JSON');
                    return;
                }
            }
            
            // Add authentication
            if (authType && authToken) {
                if (authType === 'bearer') {
                    headers['Authorization'] = 'Bearer ' + authToken;
                } else if (authType === 'basic') {
                    headers['Authorization'] = 'Basic ' + btoa(authToken);
                } else if (authType === 'apikey') {
                    headers['X-API-Key'] = authToken;
                }
            }
            
            // Parse parameters
            let params = {};
            if (paramsText) {
                try {
                    params = JSON.parse(paramsText);
                } catch (e) {
                    alert('Invalid parameters JSON');
                    return;
                }
            }
            
            // Build URL with parameters
            let requestUrl = url;
            if (Object.keys(params).length > 0) {
                const urlParams = new URLSearchParams(params);
                requestUrl += '?' + urlParams.toString();
            }
            
            // Prepare request options
            const options = {
                method: method,
                headers: headers
            };
            
            // Add body for non-GET requests
            if (method !== 'GET' && bodyText) {
                try {
                    options.body = bodyText;
                    headers['Content-Type'] = 'application/json';
                } catch (e) {
                    alert('Invalid body JSON');
                    return;
                }
            }
            
            // Show loading
            const container = document.getElementById('response-container');
            container.innerHTML = '<p style="text-align: center;">⏳ Sending request...</p>';
            
            // Send request
            const startTime = Date.now();
            fetch(requestUrl, options)
                .then(response => {
                    const endTime = Date.now();
                    const duration = endTime - startTime;
                    
                    return response.text().then(text => {
                        let responseBody;
                        try {
                            responseBody = JSON.stringify(JSON.parse(text), null, 2);
                        } catch (e) {
                            responseBody = text;
                        }
                        
                        displayResponse(response.status, response.headers, responseBody, duration);
                    });
                })
                .catch(error => {
                    displayError(error.message);
                });
        }
        
        function displayResponse(status, headers, body, duration) {
            const statusClass = status >= 200 && status < 300 ? 'success' : 'error';
            const statusColor = status >= 200 && status < 300 ? '#28a745' : '#dc3545';
            
            let headersHtml = '';
            if (headers) {
                for (let [key, value] of headers.entries()) {
                    headersHtml += key + ': ' + value + '\\n';
                }
            }
            
            const html = 
                '<div class="response-section">' +
                    '<div class="response-header">' +
                        '<strong>Status:</strong> ' +
                        '<span style="color: ' + statusColor + '; font-weight: bold;">' + status + '</span>' +
                        '<span style="float: right;">⏱️ ' + duration + 'ms</span>' +
                    '</div>' +
                '</div>' +
                
                '<div class="response-section">' +
                    '<div class="response-header">📋 Headers</div>' +
                    '<div class="response-body">' +
                        '<pre>' + headersHtml + '</pre>' +
                    '</div>' +
                '</div>' +
                
                '<div class="response-section">' +
                    '<div class="response-header">📄 Body</div>' +
                    '<div class="response-body">' +
                        '<pre>' + body + '</pre>' +
                    '</div>' +
                '</div>';
            
            document.getElementById('response-container').innerHTML = html;
        }
        
        function displayError(error) {
            const html = 
                '<div class="response-section">' +
                    '<div class="response-header" style="background: #f8d7da; color: #721c24;">❌ Error</div>' +
                    '<div class="response-body" style="background: #f8d7da;">' +
                        '<pre>' + error + '</pre>' +
                    '</div>' +
                '</div>';
            
            document.getElementById('response-container').innerHTML = html;
        }
        
        // Load available endpoints from OpenAPI spec
        fetch('{{.BasePath}}/openapi.json')
            .then(response => response.json())
            .then(spec => {
                populateEndpoints(spec);
            })
            .catch(error => {
                console.log('Could not load OpenAPI spec:', error);
            });
        
        function populateEndpoints(spec) {
            const endpointList = document.getElementById('endpoints');
            endpointList.innerHTML = '';
            
            for (const [path, pathItem] of Object.entries(spec.paths)) {
                for (const [method, operation] of Object.entries(pathItem)) {
                    if (['get', 'post', 'put', 'delete', 'patch', 'options', 'head'].includes(method)) {
                        const item = document.createElement('div');
                        item.className = 'endpoint-item';
                        item.onclick = () => selectEndpoint(method.toUpperCase(), path);
                        
                        item.innerHTML = 
                            '<span class="method-badge method-' + method + '">' + method.toUpperCase() + '</span>' +
                            path +
                            (operation.summary ? '<br><small>' + operation.summary + '</small>' : '');
                        
                        endpointList.appendChild(item);
                    }
                }
            }
        }
    </script>
</body>
</html>`

const codeGenTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{.Title}} - Code Generator</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #1f1f1f; color: white; padding: 20px; }
        .header h1 { margin: 0; }
        .header nav { margin-top: 10px; }
        .header a { color: #61affe; text-decoration: none; margin-right: 20px; }
        
        .content { padding: 20px; }
        .generator-form { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
        .form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .form-group { margin-bottom: 20px; }
        .form-group label { display: block; margin-bottom: 8px; font-weight: bold; color: #333; }
        .form-control { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
        .form-control:focus { outline: none; border-color: #61affe; box-shadow: 0 0 0 2px rgba(97, 175, 254, 0.1); }
        
        .language-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .language-card { background: #f8f9fa; border: 2px solid #dee2e6; border-radius: 8px; padding: 15px; cursor: pointer; transition: all 0.3s; }
        .language-card:hover { border-color: #61affe; background: #e3f2fd; }
        .language-card.selected { border-color: #61affe; background: #e3f2fd; }
        .language-card h4 { margin: 0 0 5px 0; color: #333; }
        .language-card p { margin: 0; color: #666; font-size: 0.9em; }
        
        .btn { padding: 12px 24px; background: #61affe; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        .btn:hover { background: #4e90d9; }
        .btn:disabled { background: #ccc; cursor: not-allowed; }
        
        .generated-code { background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .code-header { background: #f8f9fa; padding: 15px; border-bottom: 1px solid #dee2e6; display: flex; justify-content: between; align-items: center; }
        .code-content { padding: 20px; }
        .code-block { background: #2d3748; color: #e2e8f0; padding: 20px; border-radius: 4px; overflow-x: auto; font-family: 'Monaco', 'Menlo', monospace; }
        
        .download-section { text-align: center; margin: 20px 0; }
        .download-btn { background: #28a745; }
        .download-btn:hover { background: #218838; }
        
        .tabs { border-bottom: 1px solid #dee2e6; margin-bottom: 20px; }
        .tab { display: inline-block; padding: 10px 20px; cursor: pointer; border-bottom: 2px solid transparent; }
        .tab.active { border-bottom-color: #61affe; color: #61affe; }
        .tab-content { display: none; }
        .tab-content.active { display: block; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
            <nav>
                <a href="{{.BasePath}}">📖 Documentation</a>
                <a href="{{.BasePath}}/versions">📋 Versions</a>
                <a href="{{.BasePath}}/playground">🎮 Playground</a>
            </nav>
        </div>
        
        <div class="content">
            <div class="generator-form">
                <h2>⚡ Generate API Client Code</h2>
                <p>Generate client libraries and code samples for your API in multiple programming languages.</p>
                
                <div class="form-grid">
                    <div class="form-group">
                        <label>API Specification</label>
                        <select class="form-control" id="specUrl">
                            {{range .AvailableSpecs}}
                            <option value="{{.url}}">{{.name}}</option>
                            {{end}}
                        </select>
                    </div>
                    
                    <div class="form-group">
                        <label>Package Name</label>
                        <input type="text" class="form-control" id="packageName" placeholder="my-api-client" value="api-client">
                    </div>
                </div>
                
                <div class="form-group">
                    <label>Select Programming Language</label>
                    <div class="language-grid" id="languageGrid">
                        {{range .Languages}}
                        <div class="language-card" data-language="{{.id}}" onclick="selectLanguage('{{.id}}')">
                            <h4>{{.name}}</h4>
                            <p>{{.description}}</p>
                        </div>
                        {{end}}
                    </div>
                </div>
                
                <div class="tabs">
                    <div class="tab active" onclick="switchTab('client')">Client Library</div>
                    <div class="tab" onclick="switchTab('examples')">Code Examples</div>
                    <div class="tab" onclick="switchTab('models')">Models</div>
                </div>
                
                <div id="client-tab" class="tab-content active">
                    <button class="btn" onclick="generateCode('client')" id="generateBtn">
                        🔧 Generate Client Library
                    </button>
                </div>
                
                <div id="examples-tab" class="tab-content">
                    <button class="btn" onclick="generateCode('examples')" id="generateExamplesBtn">
                        📝 Generate Code Examples
                    </button>
                </div>
                
                <div id="models-tab" class="tab-content">
                    <button class="btn" onclick="generateCode('models')" id="generateModelsBtn">
                        🏗️ Generate Data Models
                    </button>
                </div>
            </div>
            
            <div class="generated-code" id="codeOutput" style="display: none;">
                <div class="code-header">
                    <h3 id="codeTitle">Generated Code</h3>
                    <div>
                        <button class="btn download-btn" onclick="downloadCode()">📥 Download</button>
                        <button class="btn" onclick="copyToClipboard()">📋 Copy</button>
                    </div>
                </div>
                <div class="code-content">
                    <div class="code-block" id="codeBlock"></div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        let selectedLanguage = null;
        let generatedCode = null;
        let codeType = null;
        
        function selectLanguage(languageId) {
            // Remove previous selection
            document.querySelectorAll('.language-card').forEach(card => card.classList.remove('selected'));
            
            // Add selection to clicked card
            document.querySelector('[data-language="' + languageId + '"]').classList.add('selected');
            selectedLanguage = languageId;
        }
        
        function switchTab(tabName) {
            // Hide all tab contents
            document.querySelectorAll('.tab-content').forEach(tab => tab.classList.remove('active'));
            document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
            
            // Show selected tab
            document.getElementById(tabName + '-tab').classList.add('active');
            event.target.classList.add('active');
        }
        
        function generateCode(type) {
            if (!selectedLanguage) {
                alert('Please select a programming language first');
                return;
            }
            
            const specUrl = document.getElementById('specUrl').value;
            const packageName = document.getElementById('packageName').value;
            
            const btn = event.target;
            const originalText = btn.textContent;
            btn.textContent = '⏳ Generating...';
            btn.disabled = true;
            
            codeType = type;
            
            // Mock code generation - in real implementation, this would call the backend
            setTimeout(() => {
                const mockCode = generateMockCode(selectedLanguage, type, packageName);
                displayGeneratedCode(mockCode, selectedLanguage, type);
                
                btn.textContent = originalText;
                btn.disabled = false;
            }, 2000);
        }
        
        function generateMockCode(language, type, packageName) {
            const examples = {
                javascript: {
                    client: '// ' + packageName + ' - JavaScript API Client
import ApiClient from './api-client';

const client = new ApiClient({
    baseURL: 'https://api.example.com',
    apiKey: 'your-api-key'
});

// Get all users
const users = await client.users.list();

// Create a new user
const newUser = await client.users.create({
    name: 'John Doe',
    email: 'john@example.com'
});

// Get user by ID
const user = await client.users.get(123);

// Update user
const updatedUser = await client.users.update(123, {
    name: 'Jane Doe'
});

// Delete user
await client.users.delete(123);',
                    examples: '// Example usage of ' + packageName + '\n' +
'import { ApiClient, User } from \'' + packageName + '\';

const client = new ApiClient('https://api.example.com', 'your-api-key');

async function examples() {
    try {
        // List users with pagination
        const users = await client.users.list({ page: 1, limit: 10 });
        console.log('Users:', users.data);
        
        // Create user
        const userData: User = {
            name: 'John Doe',
            email: 'john@example.com'
        };
        const newUser = await client.users.create(userData);
        console.log('Created user:', newUser.data);
        
    } catch (error) {
        console.error('API Error:', error);
    }
}',
                    models: '// Data models for ' + packageName + '\n' +
'export interface User {
    id: number;
    name: string;
    email: string;
    created_at: string;
    updated_at: string;
}

export interface ApiResponse<T> {
    success: boolean;
    data: T;
    message?: string;
}

export interface PaginatedResponse<T> {
    success: boolean;
    data: T[];
    meta: {
        current_page: number;
        per_page: number;
        total: number;
        total_pages: number;
        has_next: boolean;
        has_prev: boolean;
    };
}'
                },
                python: {
                    client: '# ' + packageName + ' - Python API Client\n' +
import requests
from typing import Dict, Any, Optional

class ApiClient:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        })
    
    def get_users(self, page: int = 1, limit: int = 10) -> Dict[str, Any]:
        response = self.session.get(
            f'{self.base_url}/api/v1/users',
            params={'page': page, 'limit': limit}
        )
        return response.json()
    
    def create_user(self, user_data: Dict[str, Any]) -> Dict[str, Any]:
        response = self.session.post(
            f'{self.base_url}/api/v1/users',
            json=user_data
        )
        return response.json()
    
    def get_user(self, user_id: int) -> Dict[str, Any]:
        response = self.session.get(f'{self.base_url}/api/v1/users/{user_id}')
        return response.json()
    
    def update_user(self, user_id: int, user_data: Dict[str, Any]) -> Dict[str, Any]:
        response = self.session.put(
            f'{self.base_url}/api/v1/users/{user_id}',
            json=user_data
        )
        return response.json()
    
    def delete_user(self, user_id: int) -> None:
        self.session.delete(f\'{self.base_url}/api/v1/users/{user_id}\')',
                    examples: '# Example usage of ' + packageName + '\n' +
from api_client import ApiClient

client = ApiClient('https://api.example.com', 'your-api-key')

# List users
users = client.get_users(page=1, limit=10)
print(f"Found {len(users['data'])} users")

# Create user
new_user_data = {
    'name': 'John Doe',
    'email': 'john@example.com'
}
new_user = client.create_user(new_user_data)
print(f"Created user: {new_user['data']['name']}")

# Get specific user
user = client.get_user(new_user['data']['id'])
print(f"User details: {user['data']}")

# Update user
updated_data = {'name': 'Jane Doe'}
updated_user = client.update_user(user['data']['id'], updated_data)
print(f"Updated user: {updated_user['data']['name']}")

# Delete user
client.delete_user(user['data']['id'])
print("User deleted")',
                    models: '# Data models for ' + packageName + '\n' +
from dataclasses import dataclass
from typing import Optional, List
from datetime import datetime

@dataclass
class User:
    id: int
    name: str
    email: str
    created_at: datetime
    updated_at: datetime

@dataclass
class ApiResponse:
    success: bool
    data: any
    message: Optional[str] = None

@dataclass
class PaginationMeta:
    current_page: int
    per_page: int
    total: int
    total_pages: int
    has_next: bool
    has_prev: bool

@dataclass
class PaginatedResponse:
    success: bool
    data: List[any]
    meta: PaginationMeta'
                },
                go: {
                    client: '// Go API Client\npackage client\n\ntype ApiClient struct {\n    BaseURL string\n    APIKey  string\n}',
                    examples: '// Go Examples\nclient := &ApiClient{\n    BaseURL: "https://api.example.com",\n    APIKey:  "your-key",\n}',
                    models: 'type User struct {\n    ID    int\n    Name  string\n    Email string\n}'
                }
            };
            
            return (examples[language] && examples[language][type]) ? examples[language][type] : 'Code generation not available for this combination.';
        }
        
        function displayGeneratedCode(code, language, type) {
            generatedCode = code;
            
            const output = document.getElementById('codeOutput');
            const title = document.getElementById('codeTitle');
            const codeBlock = document.getElementById('codeBlock');
            
            title.textContent = 'Generated ' + type + ' (' + language + ')';
            codeBlock.textContent = code;
            output.style.display = 'block';
            
            // Scroll to code output
            output.scrollIntoView({ behavior: 'smooth' });
        }
        
        function copyToClipboard() {
            if (generatedCode) {
                navigator.clipboard.writeText(generatedCode).then(() => {
                    alert('Code copied to clipboard!');
                });
            }
        }
        
        function downloadCode() {
            if (generatedCode && selectedLanguage) {
                const extensions = {
                    javascript: 'js',
                    python: 'py',
                    go: 'go',
                    java: 'java',
                    csharp: 'cs',
                    php: 'php'
                };
                
                const extension = extensions[selectedLanguage] || 'txt';
                const filename = 'api-client-' + codeType + '.' + extension;
                
                const blob = new Blob([generatedCode], { type: 'text/plain' });
                const url = URL.createObjectURL(blob);
                
                const a = document.createElement('a');
                a.href = url;
                a.download = filename;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
            }
        }
        
        // Select first language by default
        if (document.querySelector('.language-card')) {
            const firstLanguage = document.querySelector('.language-card').getAttribute('data-language');
            selectLanguage(firstLanguage);
        }
    </script>
</body>
</html>`