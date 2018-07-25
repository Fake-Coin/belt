function wsConnect(onmsg) {
	const base = window.location.host + window.location.pathname;

	var wsProto = "ws://";
	if (window.location.protocol == "https:") {
		wsProto = "wss://";
	}

	let ws = new WebSocket(wsProto + base + "/ws");

	ws.onopen = function(evt) {
		$("#page-dimmer").removeClass("active");
	}

	ws.onmessage = onmsg;

	ws.onclose = (evt) => {
		$("#page-dimmer").addClass("active");

		setTimeout(() => wsConnect(), 5000);
		console.log("connection closed. retrying in 5 seconds.");
	};

	ws.onerror = (evt) => {
		console.log("ERROR:", evt);
		$("#page-dimmer").addClass("active");
	};
}
