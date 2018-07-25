const COIN = 100000000;

function ProgressManager() {
	const options = new Map();
	return {
		totalValue: function() {
			var acc = 0;
			options.forEach((v) => { acc += v.value });
			return acc;
		},
		addOption: function(id, val) {
			options.set(id, {
				elm: $("#option-"+id).find(".progress"),
				value: val
			});
		},
		addTx: function(id, tx) {
			if (options.has(id)) {
				options.get(id).value += tx.value;
			}
		},
		updateAll: function() {
			const total = Math.ceil(this.totalValue()/COIN);
			options.forEach((v) => { 
				$(v.elm).progress({
					value: Math.floor(v.value/COIN),
					total: total,
					label: 'ratio',
					text: {
						ratio: '{value}\uD835\uDE41',
						active: '{value}\uD835\uDE41',
						success: '{value}\uD835\uDE41'
					}
				})
			});
		},
	};
}

function showModal(optName, optID) {
	$('#qr-display').hide();
	$('#payout-form').show();
	$('#payout-form').removeClass("hidden");
	$('#qr-display').removeClass("visible");

	$("#modal-alert").empty()
	$('#modal-header').text(optName);
	$('#payout-optionID').val(optID);

	$('.ui.modal').modal('show');
}

function showModalAlert(err) {
	const header = $('<div class="header">');
	header.text("Error getting bid address");
	const info = $("<p>");
	info.text(err);
	const closeIcon = $('<i class="close icon"></i>')

	const msg = $('<div class="ui small negative message">');
	msg.append(closeIcon, header, info);
	$("#modal-alert").append(msg);
	
	closeIcon.click(() => msg.remove())
}

function submitPayoutForm() {
	const data = 'optionID='+$('#payout-optionID').val()+'&payAddr='+$('#payout-address').val();
	const req = fetch(window.location.pathname + '/getaddr', {
		method: 'POST',
		headers: {'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'},
		body: data
	});

	$('#payout-form').transition({
		animation:'slide down',
		onComplete: function() {
			$('#qr-spinner').toggleClass("active");

			req.then(function(response) {
				$('#qr-spinner').toggleClass("active");

				if (!response.ok) {
					$('#payout-form').transition("slide down");
					response.text().then(function (text) {
						showModalAlert(text)
					});

					return;
				}
				$('#payout-form').hide();

				$('#qr-display').transition('slide down');

				return response.json();
			}).catch(function(error) {
				$('#qr-spinner').toggleClass("active");
				$('#payout-form').transition("slide down");
				modal.showModalAlert(error)
			}).then(function(msg) {
				console.log(msg);
				$('#qr-img').attr('src', "https://api.fakco.in/qr/"+ msg.watchaddr +"?size=256");
				$('#qr-link').attr("href", "fakecoin:"+msg.watchaddr);
				$('#qr-link').text(msg.watchaddr);
			});
		}
	});
}
