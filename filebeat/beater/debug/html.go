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
        .json-key {
            color: #9cdcfe;
        }
        .json-string {
            color: #ce9178;
        }
        .json-number {
            color: #b5cea8;
        }
        .json-boolean {
            color: #569cd6;
        }
        .json-null {
            color: #569cd6;
            font-style: italic;
        }
        .json-punctuation {
            color: #d4d4d4;
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
                    <label for="autoRefresh">Auto-refresh:</label>
                    <select id="refreshIntervalSelect" onchange="changeRefreshInterval()">
                        <option value="500">500ms</option>
                        <option value="1000" selected>1s</option>
                        <option value="2000">2s</option>
                        <option value="5000">5s</option>
                        <option value="10000">10s</option>
                        <option value="30000">30s</option>
                        <option value="60000">1m</option>
                    </select>
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
        let refreshIntervalMS = 1000; // Default: 1s

        function toggleAutoRefresh() {
            const checkbox = document.getElementById('autoRefresh');
            if (checkbox.checked) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        }

        function changeRefreshInterval() {
            const select = document.getElementById('refreshIntervalSelect');
            refreshIntervalMS = parseInt(select.value);
            // Restart auto-refresh if it's currently enabled
            const checkbox = document.getElementById('autoRefresh');
            if (checkbox && checkbox.checked) {
                startAutoRefresh();
            }
        }

        function startAutoRefresh() {
            stopAutoRefresh();
            autoRefreshInterval = setInterval(function() {
                loadPage(currentPage);
            }, refreshIntervalMS);
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
            let jsonStr;
            
            if (value === null || value === undefined) {
                jsonStr = String(value);
            } else if (typeof value === 'object') {
                try {
                    jsonStr = JSON.stringify(value, null, 2);
                } catch (e) {
                    return escapeHtml(String(value));
                }
            } else if (typeof value === 'string') {
                try {
                    const obj = JSON.parse(value);
                    jsonStr = JSON.stringify(obj, null, 2);
                } catch (e) {
                    return escapeHtml(value);
                }
            } else {
                jsonStr = String(value);
            }
            
            return highlightJSON(jsonStr);
        }
        
        function highlightJSON(jsonStr) {
            // Escape HTML first
            let html = escapeHtml(jsonStr);
            
            // Track positions that are already wrapped to avoid double-wrapping
            const wrapped = new Array(html.length).fill(false);
            
            function wrap(start, end, className) {
                for (let i = start; i < end; i++) {
                    wrapped[i] = true;
                }
                return '<span class="' + className + '">' + html.substring(start, end) + '</span>';
            }
            
            let result = '';
            let lastPos = 0;
            
            // Process in order: keys first, then strings, then other tokens
            // This avoids conflicts
            
            // Step 1: Highlight keys (quoted strings followed by colon)
            const keyRegex = /"([^"\\]|\\.)*"\s*:/g;
            const keyMatches = [];
            let match;
            while ((match = keyRegex.exec(html)) !== null) {
                keyMatches.push({
                    start: match.index,
                    end: match.index + match[0].length,
                    text: match[0]
                });
            }
            
            // Step 2: Highlight string values (quoted strings not already marked as keys)
            const stringRegex = /"([^"\\]|\\.)*"/g;
            const stringMatches = [];
            while ((match = stringRegex.exec(html)) !== null) {
                // Check if this string is already a key
                let isKey = false;
                for (let i = 0; i < keyMatches.length; i++) {
                    if (match.index >= keyMatches[i].start && match.index < keyMatches[i].end) {
                        isKey = true;
                        break;
                    }
                }
                if (!isKey) {
                    stringMatches.push({
                        start: match.index,
                        end: match.index + match[0].length,
                        text: match[0]
                    });
                }
            }
            
            // Step 3: Highlight numbers, booleans, null (avoiding strings)
            const numberRegex = /-?\d+\.?\d*/g;
            const numberMatches = [];
            while ((match = numberRegex.exec(html)) !== null) {
                // Check if inside a string
                let inString = false;
                for (let i = 0; i < keyMatches.length; i++) {
                    if (match.index >= keyMatches[i].start && match.index < keyMatches[i].end) {
                        inString = true;
                        break;
                    }
                }
                if (!inString) {
                    for (let i = 0; i < stringMatches.length; i++) {
                        if (match.index >= stringMatches[i].start && match.index < stringMatches[i].end) {
                            inString = true;
                            break;
                        }
                    }
                }
                if (!inString) {
                    numberMatches.push({
                        start: match.index,
                        end: match.index + match[0].length,
                        text: match[0]
                    });
                }
            }
            
            const boolRegex = /\b(true|false|null)\b/g;
            const boolMatches = [];
            while ((match = boolRegex.exec(html)) !== null) {
                // Check if inside a string
                let inString = false;
                for (let i = 0; i < keyMatches.length; i++) {
                    if (match.index >= keyMatches[i].start && match.index < keyMatches[i].end) {
                        inString = true;
                        break;
                    }
                }
                if (!inString) {
                    for (let i = 0; i < stringMatches.length; i++) {
                        if (match.index >= stringMatches[i].start && match.index < stringMatches[i].end) {
                            inString = true;
                            break;
                        }
                    }
                }
                if (!inString) {
                    boolMatches.push({
                        start: match.index,
                        end: match.index + match[0].length,
                        text: match[0],
                        isNull: match[0] === 'null'
                    });
                }
            }
            
            // Combine all matches and sort by position
            const allMatches = [];
            keyMatches.forEach(function(m) { allMatches.push({...m, type: 'key'}); });
            stringMatches.forEach(function(m) { allMatches.push({...m, type: 'string'}); });
            numberMatches.forEach(function(m) { allMatches.push({...m, type: 'number'}); });
            boolMatches.forEach(function(m) { allMatches.push({...m, type: m.isNull ? 'null' : 'boolean'}); });
            
            allMatches.sort(function(a, b) { return a.start - b.start; });
            
            // Build result string
            for (let i = 0; i < allMatches.length; i++) {
                const m = allMatches[i];
                
                // Add text before this match
                if (m.start > lastPos) {
                    // Check for punctuation in the gap
                    let gap = html.substring(lastPos, m.start);
                    gap = gap.replace(/([{}[\],:])/g, '<span class="json-punctuation">$1</span>');
                    result += gap;
                }
                
                // Add the highlighted match
                const className = 'json-' + m.type;
                result += '<span class="' + className + '">' + m.text + '</span>';
                lastPos = m.end;
            }
            
            // Add remaining text
            if (lastPos < html.length) {
                let remaining = html.substring(lastPos);
                remaining = remaining.replace(/([{}[\],:])/g, '<span class="json-punctuation">$1</span>');
                result += remaining;
            }
            
            return result;
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
