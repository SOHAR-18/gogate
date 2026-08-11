package admin

import (
	"fmt"
	"html"
	"net/http"
	"time"
)

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	routes := h.routesConfig.Routes

	routeRows := ""
	instanceRows := ""

	totalInstances := 0
	healthyInstances := 0
	healthyRoutes := 0

	// Build route table
	for _, route := range routes {
		lb := h.balancers[route.Path]

		instanceCount := 0
		healthyCount := 0

		if lb != nil {
			for _, inst := range lb.GetAll() {
				instanceCount++
				totalInstances++

				if inst.IsHealthy() {
					healthyCount++
					healthyInstances++
				}
			}
		}

		if healthyCount > 0 {
			healthyRoutes++
		}

		cb := h.cbManager.GetOrCreate(route.Path)
		cbState := string(cb.State())

		cbColor := "#1D9E75"

		switch cbState {
		case "open":
			cbColor = "#E53E3E"
		case "half-open":
			cbColor = "#EF9F27"
		}

		protected := "No"
		protectedColor := "#6B7280"

		if route.Protected {
			protected = "Yes"
			protectedColor = "#7C3AED"
		}

		statusText := "No healthy instances"
		statusColor := "#E53E3E"

		if healthyCount > 0 {
			statusText = "Healthy"
			statusColor = "#1D9E75"
		}

		routeRows += fmt.Sprintf(`
			<tr>
				<td>
					<strong>%s</strong>
				</td>

				<td>
					<span class="badge" style="color:%s">
						%s
					</span>
				</td>

				<td>
					<span class="badge" style="color:%s">
						%s
					</span>
				</td>

				<td>
					<span style="font-weight:600">
						%d / %d
					</span>
				</td>

				<td>
					%d req / %ds
				</td>

				<td>
					<span class="status-dot" style="background:%s"></span>
					<span style="color:%s;font-weight:600">
						%s
					</span>
				</td>

				<td>
					<span style="color:%s;font-weight:600">
						%s
					</span>
				</td>
			</tr>
		`,
			html.EscapeString(route.Path),
			protectedColor,
			protected,
			statusColor,
			statusText,
			healthyCount,
			instanceCount,
			route.RateLimit,
			route.RateWindow,
			cbColor,
			cbColor,
			cbState,
			statusColor,
			statusText,
		)
	}

	// Build instance table
	for _, route := range routes {
		lb := h.balancers[route.Path]

		if lb == nil {
			continue
		}

		for _, inst := range lb.GetAll() {
			healthColor := "#1D9E75"
			healthText := "Healthy"

			if !inst.IsHealthy() {
				healthColor = "#E53E3E"
				healthText = "Unhealthy"
			}

			instanceRows += fmt.Sprintf(`
				<tr>
					<td>
						<strong>%s</strong>
					</td>

					<td class="mono">
						%s
					</td>

					<td>
						<span class="status-dot" style="background:%s"></span>
						<span style="color:%s;font-weight:600">
							%s
						</span>
					</td>

					<td>
						<span class="route-label">
							%s
						</span>
					</td>
				</tr>
			`,
				html.EscapeString(route.Path),
				html.EscapeString(inst.RawURL),
				healthColor,
				healthColor,
				healthText,
				html.EscapeString(route.Path),
			)
		}
	}

	// Handle empty route list
	if routeRows == "" {
		routeRows = `
			<tr>
				<td colspan="7" class="empty">
					No routes configured.
				</td>
			</tr>
		`
	}

	// Handle empty instance list
	if instanceRows == "" {
		instanceRows = `
			<tr>
				<td colspan="4" class="empty">
					No upstream instances discovered.
				</td>
			</tr>
		`
	}

	healthPercentage := 0

	if totalInstances > 0 {
		healthPercentage = (healthyInstances * 100) / totalInstances
	}

	htmlPage := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">

<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">

	<title>GoGate Admin Dashboard</title>

	<meta http-equiv="refresh" content="30">

	<style>
		* {
			box-sizing: border-box;
		}

		body {
			margin: 0;
			background: #f5f7fa;
			color: #1f2937;
			font-family:
				Inter,
				Segoe UI,
				Roboto,
				Arial,
				sans-serif;
		}

		.header {
			background: #111827;
			color: white;
			padding: 22px 32px;
			display: flex;
			align-items: center;
			justify-content: space-between;
			box-shadow: 0 2px 8px rgba(0,0,0,0.15);
		}

		.header-left h1 {
			margin: 0;
			font-size: 24px;
			font-weight: 700;
		}

		.header-left p {
			margin: 6px 0 0;
			color: #9ca3af;
			font-size: 13px;
		}

		.header-right {
			display: flex;
			align-items: center;
			gap: 10px;
		}

		.live {
			display: flex;
			align-items: center;
			gap: 7px;
			padding: 7px 12px;
			background: rgba(29,158,117,0.15);
			border-radius: 20px;
			color: #6ee7b7;
			font-size: 12px;
			font-weight: 600;
		}

		.live-dot {
			width: 8px;
			height: 8px;
			border-radius: 50%%;
			background: #1D9E75;
			box-shadow: 0 0 8px rgba(29,158,117,0.8);
		}

		.refresh-btn {
			border: 0;
			background: #374151;
			color: white;
			padding: 8px 14px;
			border-radius: 6px;
			cursor: pointer;
			font-size: 12px;
		}

		.refresh-btn:hover {
			background: #4b5563;
		}

		.container {
			max-width: 1400px;
			margin: 0 auto;
			padding: 28px 32px;
		}

		.cards {
			display: grid;
			grid-template-columns: repeat(4, 1fr);
			gap: 18px;
			margin-bottom: 28px;
		}

		.card {
			background: white;
			border: 1px solid #e5e7eb;
			border-radius: 10px;
			padding: 20px;
			box-shadow: 0 1px 3px rgba(0,0,0,0.05);
		}

		.card-title {
			color: #6b7280;
			font-size: 12px;
			font-weight: 600;
			text-transform: uppercase;
			letter-spacing: 0.5px;
		}

		.card-value {
			font-size: 30px;
			font-weight: 700;
			margin-top: 8px;
		}

		.card-subtitle {
			margin-top: 5px;
			font-size: 12px;
			color: #9ca3af;
		}

		.green {
			color: #1D9E75;
		}

		.red {
			color: #E53E3E;
		}

		.purple {
			color: #7C3AED;
		}

		.blue {
			color: #2563EB;
		}

		.section {
			background: white;
			border: 1px solid #e5e7eb;
			border-radius: 10px;
			margin-bottom: 24px;
			overflow: hidden;
			box-shadow: 0 1px 3px rgba(0,0,0,0.04);
		}

		.section-header {
			padding: 18px 20px;
			border-bottom: 1px solid #e5e7eb;
			display: flex;
			align-items: center;
			justify-content: space-between;
		}

		.section-header h2 {
			margin: 0;
			font-size: 16px;
		}

		.section-header span {
			color: #9ca3af;
			font-size: 12px;
		}

		table {
			width: 100%%;
			border-collapse: collapse;
		}

		th {
			background: #f9fafb;
			color: #6b7280;
			font-size: 11px;
			text-transform: uppercase;
			letter-spacing: 0.5px;
			text-align: left;
			padding: 12px 18px;
			border-bottom: 1px solid #e5e7eb;
		}

		td {
			padding: 14px 18px;
			border-bottom: 1px solid #f0f1f3;
			font-size: 13px;
		}

		tr:last-child td {
			border-bottom: none;
		}

		tr:hover td {
			background: #fafafa;
		}

		.badge {
			font-size: 12px;
			font-weight: 600;
		}

		.status-dot {
			display: inline-block;
			width: 8px;
			height: 8px;
			border-radius: 50%%;
			margin-right: 6px;
		}

		.route-label {
			display: inline-block;
			padding: 4px 8px;
			background: #f3f4f6;
			border-radius: 5px;
			font-size: 11px;
			font-family: monospace;
		}

		.mono {
			font-family: Consolas, Monaco, monospace;
			font-size: 12px;
		}

		.empty {
			text-align: center;
			padding: 35px;
			color: #9ca3af;
		}

		.progress {
			width: 100%%;
			height: 7px;
			background: #e5e7eb;
			border-radius: 10px;
			overflow: hidden;
			margin-top: 10px;
		}

		.progress-bar {
			height: 100%%;
			background: #1D9E75;
			border-radius: 10px;
			width: %d%%;
		}

		.footer {
			text-align: center;
			color: #9ca3af;
			font-size: 11px;
			padding: 10px 0 25px;
		}

		@media (max-width: 900px) {
			.cards {
				grid-template-columns: repeat(2, 1fr);
			}

			.container {
				padding: 20px;
			}

			.section {
				overflow-x: auto;
			}

			table {
				min-width: 800px;
			}
		}

		@media (max-width: 550px) {
			.cards {
				grid-template-columns: 1fr;
			}

			.header {
				padding: 18px;
			}

			.header-right {
				display: none;
			}
		}
	</style>
</head>

<body>

	<header class="header">

		<div class="header-left">
			<h1>GoGate Admin Dashboard</h1>
			<p>API Gateway • Service Discovery • Health Monitoring</p>
		</div>

		<div class="header-right">

			<div class="live">
				<span class="live-dot"></span>
				LIVE
			</div>

			<button class="refresh-btn" onclick="location.reload()">
				Refresh
			</button>

		</div>

	</header>

	<main class="container">

		<div class="cards">

			<div class="card">
				<div class="card-title">
					Routes
				</div>

				<div class="card-value blue">
					%d
				</div>

				<div class="card-subtitle">
					%d healthy routes
				</div>
			</div>

			<div class="card">
				<div class="card-title">
					Instances
				</div>

				<div class="card-value purple">
					%d
				</div>

				<div class="card-subtitle">
					Total discovered instances
				</div>
			</div>

			<div class="card">
				<div class="card-title">
					Healthy Instances
				</div>

				<div class="card-value green">
					%d
				</div>

				<div class="card-subtitle">
					%d%% health rate
				</div>

				<div class="progress">
					<div class="progress-bar"></div>
				</div>
			</div>

			<div class="card">
				<div class="card-title">
					Gateway Status
				</div>

				<div class="card-value green">
					Online
				</div>

				<div class="card-subtitle">
					Monitoring active
				</div>
			</div>

		</div>

		<section class="section">

			<div class="section-header">
				<h2>Routes</h2>
				<span>Gateway route configuration</span>
			</div>

			<table>

				<thead>
					<tr>
						<th>Route</th>
						<th>Protection</th>
						<th>Status</th>
						<th>Healthy / Total</th>
						<th>Rate Limit</th>
						<th>Circuit Breaker</th>
						<th>Health</th>
					</tr>
				</thead>

				<tbody>
					%s
				</tbody>

			</table>

		</section>

		<section class="section">

			<div class="section-header">
				<h2>Upstream Instances</h2>
				<span>Service discovery and health status</span>
			</div>

			<table>

				<thead>
					<tr>
						<th>Route</th>
						<th>Instance URL</th>
						<th>Health</th>
						<th>Route</th>
					</tr>
				</thead>

				<tbody>
					%s
				</tbody>

			</table>

		</section>

		<div class="footer">
			GoGate Admin Dashboard • Auto-refreshes every 30 seconds
		</div>

	</main>

</body>

</html>`,
		healthPercentage,
		len(routes),
		healthyRoutes,
		totalInstances,
		healthyInstances,
		healthPercentage,
		routeRows,
		instanceRows,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprint(w, htmlPage)
}

func (h *Handler) countHealthy() int {
	count := 0

	for _, route := range h.routesConfig.Routes {
		lb := h.balancers[route.Path]

		if lb == nil {
			continue
		}

		for _, inst := range lb.GetAll() {
			if inst.IsHealthy() {
				count++
			}
		}
	}

	return count
}

// Keep the compiler happy if time is used by future dashboard features.
var _ = time.Second
