// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// This file was contributed to by generative AI

package debug

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Filebeat Registry Debug</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: #1a1a1a;
            color: #e0e0e0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        header {
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #333;
        }
        h1 {
            color: #4a9eff;
            margin-bottom: 10px;
        }
        .controls {
            display: flex;
            gap: 15px;
            margin-bottom: 20px;
            flex-wrap: wrap;
            align-items: center;
        }
        .control-group {
            display: flex;
            gap: 10px;
            align-items: center;
        }
        label {
            color: #aaa;
            font-size: 14px;
        }
        input, button, select {
            padding: 8px 12px;
            border: 1px solid #444;
            border-radius: 4px;
            background: #2a2a2a;
            color: #e0e0e0;
            font-size: 14px;
        }
        input:focus, select:focus {
            outline: none;
            border-color: #4a9eff;
        }
        button {
            cursor: pointer;
            background: #4a9eff;
            border-color: #4a9eff;
            color: white;
            font-weight: 500;
        }
        button:hover {
            background: #3a8eef;
        }
        button:disabled {
            background: #444;
            border-color: #444;
            cursor: not-allowed;
            opacity: 0.6;
        }
        .stats {
            color: #aaa;
            font-size: 14px;
            margin-bottom: 20px;
        }
        .keys-list {
            background: #2a2a2a;
            border-radius: 8px;
            overflow: hidden;
            margin-bottom: 20px;
        }
        .key-item {
            padding: 15px;
            border-bottom: 1px solid #333;
            cursor: pointer;
            transition: background 0.2s;
        }
        .key-item:hover {
            background: #333;
        }
        .key-item:last-child {
            border-bottom: none;
        }
        .key-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .key-name {
            font-family: 'Courier New', monospace;
            color: #4a9eff;
            font-weight: 600;
            word-break: break-all;
        }
        .key-value {
            display: none;
            margin-top: 10px;
            padding: 10px;
            background: #1a1a1a;
            border-radius: 4px;
            border-left: 3px solid #4a9eff;
        }
        .key-value.expanded {
            display: block;
        }
        .json-content {
            font-family: 'Courier New', monospace;
            font-size: 12px;
            color: #e0e0e0;
            white-space: pre-wrap;
            word-wrap: break-word;
            overflow-x: auto;
        }
        .error {
            color: #ff6b6b;
        }
        .pagination {
            display: flex;
            gap: 10px;
            justify-content: center;
            align-items: center;
            margin-top: 20px;
        }
        .pagination input {
            width: 60px;
            text-align: center;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: #aaa;
        }
        .empty {
            text-align: center;
            padding: 40px;
            color: #666;
        }
        .auto-refresh {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .auto-refresh input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
        }
        @media (prefers-color-scheme: light) {
            body {
                background: #f5f5f5;
                color: #333;
            }
            .keys-list, .key-value {
                background: white;
            }
            .key-item:hover {
                background: #f0f0f0;
            }
            input, button, select {
                background: white;
                color: #333;
                border-color: #ddd;
            }
            .key-value {
                background: #f9f9f9;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Filebeat Registry Debug</h1>
            <div class="stats" id="stats">Loading...</div>
        </header>
        
        <div class="controls">
            <div class="control-group">
                <button onclick="loadPage(1)">Refresh</button>
                <div class="auto-refresh">
                    <input type="checkbox" id="autoRefresh" checked onchange="toggleAutoRefresh()">
                    <label for="autoRefresh">Auto-refresh (10s)</label>
                </div>
            </div>
            <div class="control-group">
                <label>Page:</label>
                <input type="number" id="pageInput" min="1" value="1" onchange="goToPage()">
                <span id="pageInfo">of 1</span>
            </div>
            <div class="control-group">
                <label>Page Size:</label>
                <select id="pageSizeSelect" onchange="changePageSize()">
                    <option value="25">25</option>
                    <option value="50" selected>50</option>
                    <option value="100">100</option>
                    <option value="250">250</option>
                </select>
            </div>
            <div class="control-group">
                <label>Search:</label>
                <input type="text" id="searchInput" placeholder="Filter by key..." oninput="filterKeys()">
            </div>
        </div>

        <div class="keys-list" id="keysList">
            <div class="loading">Loading...</div>
        </div>

        <div class="pagination">
            <button onclick="loadPage(currentPage - 1)" id="prevBtn" disabled>Previous</button>
            <span>Page <span id="currentPage">1</span> of <span id="totalPages">1</span></span>
            <button onclick="loadPage(currentPage + 1)" id="nextBtn" disabled>Next</button>
        </div>
    </div>

    <script>
        let currentPage = 1;
        let currentPageSize = 50;
        let allKeys = [];
        let autoRefreshInterval = null;

        function toggleAutoRefresh() {
            const checkbox = document.getElementById('autoRefresh');
            if (checkbox.checked) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        }

        function startAutoRefresh() {
            stopAutoRefresh();
            autoRefreshInterval = setInterval(() => {
                loadPage(currentPage);
            }, 10000);
        }

        function stopAutoRefresh() {
            if (autoRefreshInterval) {
                clearInterval(autoRefreshInterval);
                autoRefreshInterval = null;
            }
        }

        function changePageSize() {
            currentPageSize = parseInt(document.getElementById('pageSizeSelect').value);
            currentPage = 1;
            loadPage(1);
        }

        function goToPage() {
            const page = parseInt(document.getElementById('pageInput').value);
            if (page >= 1) {
                loadPage(page);
            }
        }

        function loadPage(page) {
            const url = '/api/keys?page=' + page + '&page_size=' + currentPageSize;
            
            fetch(url)
                .then(function(response) {
                    if (!response.ok) {
                        throw new Error('Failed to load keys');
                    }
                    return response.json();
                })
                .then(function(data) {
                    currentPage = data.page;
                    allKeys = data.keys;
                    
                    updateUI(data);
                    filterKeys();
                })
                .catch(function(error) {
                    document.getElementById('keysList').innerHTML = 
                        '<div class="empty">Error loading keys: ' + error.message + '</div>';
                });
        }

        function updateUI(data) {
            document.getElementById('stats').textContent = 
                'Total keys: ' + data.total + ' | Showing ' + data.keys.length + ' on page ' + data.page + ' of ' + data.total_pages;
            
            document.getElementById('currentPage').textContent = data.page;
            document.getElementById('totalPages').textContent = data.total_pages;
            document.getElementById('pageInput').value = data.page;
            document.getElementById('pageInfo').textContent = 'of ' + data.total_pages;
            
            document.getElementById('prevBtn').disabled = data.page <= 1;
            document.getElementById('nextBtn').disabled = data.page >= data.total_pages;
            
            renderKeys(data.keys);
        }

        function renderKeys(keys) {
            const container = document.getElementById('keysList');
            if (keys.length === 0) {
                container.innerHTML = '<div class="empty">No keys found</div>';
                return;
            }

            container.innerHTML = keys.map(function(kv, idx) {
                const valueStr = kv.error ? 
                    '<span class="error">' + escapeHtml(kv.error) + '</span>' :
                    formatJSON(kv.value);
                
                return '<div class="key-item" onclick="toggleKey(' + idx + ')">' +
                    '<div class="key-header">' +
                    '<span class="key-name">' + escapeHtml(kv.key) + '</span>' +
                    '</div>' +
                    '<div class="key-value expanded" id="key-' + idx + '">' +
                    '<div class="json-content">' + valueStr + '</div>' +
                    '</div>' +
                    '</div>';
            }).join('');

            // Store keys for filtering
            window.currentKeys = keys;
        }

        function toggleKey(idx) {
            const elem = document.getElementById('key-' + idx);
            if (elem) {
                elem.classList.toggle('expanded');
            }
        }

        function filterKeys() {
            const search = document.getElementById('searchInput').value.toLowerCase();
            if (!search) {
                renderKeys(allKeys);
                return;
            }

            const filtered = allKeys.filter(function(kv) {
                return kv.key.toLowerCase().indexOf(search) !== -1;
            });
            renderKeys(filtered);
        }

        function formatJSON(value) {
            if (value === null || value === undefined) {
                return escapeHtml(String(value));
            }
            
            // If value is already an object, stringify it directly
            if (typeof value === 'object') {
                try {
                    return escapeHtml(JSON.stringify(value, null, 2));
                } catch (e) {
                    return escapeHtml(String(value));
                }
            }
            
            // If value is a string, try to parse then stringify for formatting
            if (typeof value === 'string') {
                try {
                    const obj = JSON.parse(value);
                    return escapeHtml(JSON.stringify(obj, null, 2));
                } catch (e) {
                    return escapeHtml(value);
                }
            }
            
            // Fallback for other types
            return escapeHtml(String(value));
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Initialize
        loadPage(1);
        startAutoRefresh();
    </script>
</body>
</html>`
