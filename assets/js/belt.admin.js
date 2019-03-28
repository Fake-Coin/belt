function setBelt(optID, beltbtn) {
	if (optID === -1) {
		beltbtn = document.createElement("a");
	}
	if ($(beltbtn).hasClass("active")) {
		return;
	}
	
	const loader = $(beltbtn).next(".loader");
	loader.addClass("active");

	fetch(window.location.pathname + '/setbelt', {
		method: 'POST',
		credentials: "same-origin",
		headers: {'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'},
		body: "optionID="+optID
	}).then((response) => {
		if (!response.ok) {
			loader.removeClass("active");
			response.text().then(showBeltAlert);
			return;
		}
		return response.json();
	}).catch((error) => {
		showBeltAlert(error);
	}).then(function(msg) {
		loader.removeClass("active");
	});
}

function showBeltAlert(err) {
	const header = $('<div class="header">');
	header.text("Error setting belt");
	const info = $("<p>");
	info.text(err);
	const closeIcon = $('<i class="close icon"></i>')

	const msg = $('<div class="ui bottom attached negative message">');
	msg.append(closeIcon, header, info);
	$("#belt-containter").append(msg);
	
	closeIcon.click(() => msg.remove())
}
