// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>{{.Title}}</title>
	<style>
		* {
			box-sizing: border-box;
		}
		
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
			line-height: 1.6;
			margin: 0;
			padding: 0;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			min-height: 100vh;
		}
		
		.container {
			max-width: 1400px;
			margin: 0 auto;
			padding: 20px;
		}
		
		.main-content {
			background: white;
			border-radius: 12px;
			box-shadow: 0 8px 32px rgba(0,0,0,0.15);
			overflow: hidden;
		}
		
		.header {
			background: linear-gradient(135deg, #0f1419 0%, #2d3748 100%);
			color: white;
			padding: 30px;
			position: relative;
			overflow: hidden;
		}
		
		.header::before {
			content: '';
			position: absolute;
			top: 0;
			left: 0;
			right: 0;
			bottom: 0;
			background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><defs><pattern id="grid" width="10" height="10" patternUnits="userSpaceOnUse"><path d="M 10 0 L 0 0 0 10" fill="none" stroke="rgba(255,255,255,0.1)" stroke-width="1"/></pattern></defs><rect width="100" height="100" fill="url(%23grid)"/></svg>');
			opacity: 0.3;
		}
		
		.header-content {
			position: relative;
			z-index: 1;
		}
		
		.header h1 {
			margin: 0;
			font-size: 2.5em;
			font-weight: 700;
			background: linear-gradient(45deg, #fff, #a0aec0);
			-webkit-background-clip: text;
			-webkit-text-fill-color: transparent;
			background-clip: text;
		}
		
		.header .subtitle {
			margin-top: 8px;
			font-size: 1.1em;
			opacity: 0.8;
		}
		
		.summary {
			padding: 40px;
			background: linear-gradient(135deg, #f7fafc 0%, #edf2f7 100%);
		}
		
		.summary h2 {
			margin-top: 0;
			margin-bottom: 25px;
			color: #2d3748;
			font-size: 1.8em;
			font-weight: 600;
		}
		
		.stats {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
			gap: 25px;
			margin-top: 25px;
		}
		
		.stat {
			background: white;
			padding: 25px;
			border-radius: 12px;
			border: 1px solid #e2e8f0;
			box-shadow: 0 4px 12px rgba(0,0,0,0.05);
			position: relative;
			overflow: hidden;
			transition: transform 0.2s ease, shadow 0.2s ease;
		}
		
		.stat:hover {
			transform: translateY(-2px);
			box-shadow: 0 8px 25px rgba(0,0,0,0.1);
		}
		
		.stat::before {
			content: '';
			position: absolute;
			top: 0;
			left: 0;
			right: 0;
			height: 4px;
			background: linear-gradient(90deg, #667eea, #764ba2);
		}
		
		.stat-label {
			font-size: 0.95em;
			color: #718096;
			margin-bottom: 8px;
			font-weight: 500;
			text-transform: uppercase;
			letter-spacing: 0.5px;
		}
		
		.stat-value {
			font-size: 2.2em;
			font-weight: 700;
			color: #2d3748;
			line-height: 1;
		}
		
		.coverage-rate {
			color: {{if lt .Summary.CoverageRate 0.5}}#e53e3e{{else if lt .Summary.CoverageRate 0.8}}#dd6b20{{else}}#38a169{{end}};
		}
		
		.controls {
			padding: 40px;
			background: #f8f9fa;
			border-bottom: 1px solid #e2e8f0;
		}
		
		.controls h3 {
			margin-top: 0;
			margin-bottom: 20px;
			color: #2d3748;
			font-size: 1.5em;
			font-weight: 600;
		}
		
		.filter-row {
			display: flex;
			gap: 15px;
			flex-wrap: wrap;
			align-items: center;
		}
		
		.filter-group {
			display: flex;
			flex-direction: column;
			gap: 5px;
		}
		
		.filter-group label {
			font-size: 0.9em;
			color: #4a5568;
			font-weight: 500;
		}
		
		.filter-row input {
			padding: 12px 16px;
			border: 2px solid #e2e8f0;
			border-radius: 8px;
			font-size: 14px;
			transition: border-color 0.2s ease, box-shadow 0.2s ease;
			background: white;
		}
		
		.filter-row input:focus {
			outline: none;
			border-color: #667eea;
			box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
		}
		
		.filter-row button {
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
			border: none;
			padding: 12px 24px;
			border-radius: 8px;
			cursor: pointer;
			font-size: 14px;
			font-weight: 600;
			transition: transform 0.2s ease, box-shadow 0.2s ease;
		}
		
		.filter-row button:hover {
			transform: translateY(-1px);
			box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
		}
		
		.packages {
			padding: 40px;
		}
		
		.packages h3 {
			margin-top: 0;
			margin-bottom: 25px;
			color: #2d3748;
			font-size: 1.5em;
			font-weight: 600;
		}
		
		.package {
			border: 1px solid #e2e8f0;
			border-radius: 12px;
			margin-bottom: 20px;
			overflow: hidden;
			transition: box-shadow 0.2s ease;
		}
		
		.package:hover {
			box-shadow: 0 4px 12px rgba(0,0,0,0.08);
		}
		
		.package-header {
			background: linear-gradient(135deg, #f7fafc 0%, #edf2f7 100%);
			padding: 20px 25px;
			border-bottom: 1px solid #e2e8f0;
			cursor: pointer;
			display: flex;
			justify-content: space-between;
			align-items: center;
			transition: background 0.2s ease;
		}
		
		.package-header:hover {
			background: linear-gradient(135deg, #edf2f7 0%, #e2e8f0 100%);
		}
		
		.package-name {
			font-weight: 600;
			color: #667eea;
			font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
			font-size: 0.95em;
		}
		
		.package-coverage {
			font-weight: 600;
			font-size: 1.1em;
		}
		
		.package-content {
			display: none;
			padding: 25px;
			background: white;
		}
		
		.package-content.expanded {
			display: block;
		}
		
		.function {
			padding: 12px 0;
			border-bottom: 1px solid #f1f3f4;
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		
		.function:last-child {
			border-bottom: none;
		}
		
		.function-name {
			font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
			font-size: 0.9em;
			color: #2d3748;
		}
		
		.function-coverage {
			font-weight: 600;
			padding: 4px 8px;
			border-radius: 4px;
			font-size: 0.85em;
		}
		
		.coverage-high { 
			color: #38a169;
			background: rgba(56, 161, 105, 0.1);
		}
		.coverage-medium { 
			color: #dd6b20;
			background: rgba(221, 107, 32, 0.1);
		}
		.coverage-low { 
			color: #e53e3e;
			background: rgba(229, 62, 62, 0.1);
		}
		
		.loading {
			text-align: center;
			padding: 60px;
			color: #718096;
			font-size: 1.1em;
		}
		
		.loading::before {
			content: '';
			display: inline-block;
			width: 20px;
			height: 20px;
			border: 2px solid #e2e8f0;
			border-top: 2px solid #667eea;
			border-radius: 50%;
			animation: spin 1s linear infinite;
			margin-right: 10px;
			vertical-align: middle;
		}
		
		@keyframes spin {
			0% { transform: rotate(0deg); }
			100% { transform: rotate(360deg); }
		}
		
		.expand-icon {
			transition: transform 0.2s ease;
			font-size: 1.2em;
			color: #718096;
		}
		
		.expand-icon.expanded {
			transform: rotate(90deg);
		}
		
		.progress-bar {
			width: 100%;
			height: 8px;
			background: #e2e8f0;
			border-radius: 4px;
			overflow: hidden;
			margin-top: 8px;
		}
		
		.progress-fill {
			height: 100%;
			background: linear-gradient(90deg, #667eea, #764ba2);
			transition: width 0.3s ease;
		}
		
		@media (max-width: 768px) {
			.container {
				padding: 10px;
			}
			
			.header h1 {
				font-size: 2em;
			}
			
			.stats {
				grid-template-columns: 1fr;
			}
			
			.filter-row {
				flex-direction: column;
				align-items: stretch;
			}
			
			.package-header {
				flex-direction: column;
				align-items: flex-start;
				gap: 10px;
			}
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="main-content">
			<div class="header">
				<div class="header-content">
					<h1>{{.Title}}</h1>
					<div class="subtitle">Interactive Coverage Explorer</div>
				</div>
			</div>
			
			<div class="summary">
				<h2>Coverage Summary</h2>
				<div class="stats">
					<div class="stat">
						<div class="stat-label">Total Packages</div>
						<div class="stat-value">{{.Summary.TotalPackages}}</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: 100%"></div>
						</div>
					</div>
					<div class="stat">
						<div class="stat-label">Total Lines</div>
						<div class="stat-value">{{.Summary.TotalLines}}</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: 100%"></div>
						</div>
					</div>
					<div class="stat">
						<div class="stat-label">Covered Lines</div>
						<div class="stat-value">{{.Summary.CoveredLines}}</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: {{printf "%.0f" (mult .Summary.CoverageRate 100)}}%"></div>
						</div>
					</div>
					<div class="stat">
						<div class="stat-label">Coverage Rate</div>
						<div class="stat-value coverage-rate">{{printf "%.1f" (mult .Summary.CoverageRate 100)}}%</div>
						<div class="progress-bar">
							<div class="progress-fill" style="width: {{printf "%.0f" (mult .Summary.CoverageRate 100)}}%"></div>
						</div>
					</div>
				</div>
			</div>

			<div class="controls">
				<h3>Filter Packages</h3>
				<div class="filter-row">
					<div class="filter-group">
						<label for="filter">Package Pattern</label>
						<input type="text" id="filter" placeholder="e.g., github.com/user/*">
					</div>
					<div class="filter-group">
						<label for="minCov">Min Coverage %</label>
						<input type="number" id="minCov" placeholder="0" step="0.1" min="0" max="100">
					</div>
					<div class="filter-group">
						<label for="maxCov">Max Coverage %</label>
						<input type="number" id="maxCov" placeholder="100" step="0.1" min="0" max="100">
					</div>
					<div class="filter-group">
						<label>&nbsp;</label>
						<button onclick="loadPackages()">Apply Filter</button>
					</div>
				</div>
			</div>

			<div class="packages">
				<h3>Packages</h3>
				<div id="packages-list">
					<div class="loading">Loading packages...</div>
				</div>
			</div>
		</div>
	</div>

	<script>
		function loadPackages() {
			const filter = document.getElementById('filter').value;
			const minCov = document.getElementById('minCov').value;
			const maxCov = document.getElementById('maxCov').value;
			
			let url = '/api/packages?';
			if (filter) url += 'pattern=' + encodeURIComponent(filter) + '&';
			if (minCov) url += 'min_coverage=' + (parseFloat(minCov) / 100) + '&';
			if (maxCov) url += 'max_coverage=' + (parseFloat(maxCov) / 100) + '&';
			
			document.getElementById('packages-list').innerHTML = '<div class="loading">Loading packages...</div>';
			
			fetch(url)
				.then(response => response.json())
				.then(data => {
					const container = document.getElementById('packages-list');
					if (data.length === 0) {
						container.innerHTML = '<div class="loading">No packages match the filter criteria.</div>';
						return;
					}
					
					let html = '';
					data.forEach(pkg => {
						const coverageClass = pkg.CoverageRate > 0.8 ? 'coverage-high' : 
											pkg.CoverageRate > 0.5 ? 'coverage-medium' : 'coverage-low';
						const packageId = pkg.ImportPath.replace(/[^a-zA-Z0-9]/g, '_');
						
						html += ` + "`" + `
							<div class="package">
								<div class="package-header" onclick="togglePackage('${packageId}')">
									<span class="package-name">${pkg.ImportPath}</span>
									<div style="display: flex; align-items: center; gap: 15px;">
										<span class="package-coverage ${coverageClass}">
											${(pkg.CoverageRate * 100).toFixed(1)}% (${pkg.CoveredLines}/${pkg.TotalLines})
										</span>
										<span class="expand-icon" id="icon-${packageId}">▶</span>
									</div>
								</div>
								<div class="package-content" id="content-${packageId}">
						` + "`" + `;
						
						if (pkg.Functions && pkg.Functions.length > 0) {
							pkg.Functions.forEach(fn => {
								const fnCoverageClass = fn.CoverageRate > 0.8 ? 'coverage-high' : 
													   fn.CoverageRate > 0.5 ? 'coverage-medium' : 'coverage-low';
								html += ` + "`" + `
									<div class="function">
										<span class="function-name">${fn.Name}</span>
										<span class="function-coverage ${fnCoverageClass}">${(fn.CoverageRate * 100).toFixed(1)}%</span>
									</div>
								` + "`" + `;
							});
						} else {
							html += '<div class="function">No functions found</div>';
						}
						
						html += '</div></div>';
					});
					
					container.innerHTML = html;
				})
				.catch(error => {
					console.error('Error loading packages:', error);
					document.getElementById('packages-list').innerHTML = 
						'<div class="loading">Error loading packages. Please try again.</div>';
				});
		}

		function togglePackage(packageId) {
			const content = document.getElementById('content-' + packageId);
			const icon = document.getElementById('icon-' + packageId);
			
			if (content.classList.contains('expanded')) {
				content.classList.remove('expanded');
				icon.classList.remove('expanded');
				content.style.display = 'none';
				icon.textContent = '▶';
			} else {
				content.classList.add('expanded');
				icon.classList.add('expanded');
				content.style.display = 'block';
				icon.textContent = '▼';
			}
		}

		// Load packages on page load
		document.addEventListener('DOMContentLoaded', function() {
			loadPackages();
		});

		// Add keyboard shortcuts
		document.addEventListener('keydown', function(e) {
			if (e.ctrlKey || e.metaKey) {
				switch(e.key) {
					case 'f':
						e.preventDefault();
						document.getElementById('filter').focus();
						break;
					case 'r':
						e.preventDefault();
						loadPackages();
						break;
				}
			}
		});
	</script>
</body>
</html>`
