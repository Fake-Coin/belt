const COIN = 100000000;

function ProgressManager() {
	const options = new Map();
	return {
		totalValue: function() {
			var acc = 0;
			options.forEach((v) => { acc += v.value });
			return acc;
		},
		totalVotes: function() {
			var acc = 0;
			options.forEach((v) => { acc += v.votes });
			return acc;
		},
		addOption: function(id, val, conf, votes) {
			options.set(id, {
				betElm: $("#option-"+id).find(".optbar"),
				value: conf,
				voteElm: $("#option-"+id).find(".voteBar"),
				votes: votes
			});
		},
		addTx: function(id, tx) {
			if (!options.has(id)) {
				return;
			}
			options.get(id).value += tx.value;
			// assume confirmed for now
			// options.get(id).confirmed += tx.value;

			// if (txSet.has(tx.hash)) {
			// 	if (0 < tx.confirmations) {
			// 		options.get(id).confirmed += tx.value;
			// 		txSet.delete(tx.hash);
			// 	}
			// 	return;
			// }
			//
			// if (0 < tx.confirmations) {
			// 	options.get(id).confirmed += tx.value;
			// 	return;
			// }
			//
			// options.get(id).value += tx.value;
			// txSet.add(tx.hash)
		},
		incVotes: function(id) {
			if (!options.has(id)) {
				return;
			}
			options.get(id).votes++;
		},
		updateAll: function() {
			const tVotes = this.totalVotes();
			const total = Math.ceil(this.totalValue()/COIN);
			options.forEach((v) => {
				const val100 = Math.floor((v.value/COIN)*100);
				// const conf100 = Math.floor((v.confirmed/COIN)*100);

				var statusText = "{value}\uD835\uDE41";
				statusText += "\xa0\xa0|\xa0";
				statusText += v.votes+"\u2713";
				
				// if (val100 != conf100) {
				// 	statusText = '{value}\uD835\uDE41' + '(' + conf100/100 + '\uD835\uDE41 confirmed)';
				// }

				$(v.betElm).progress({
					value: val100/100,
					total: total,
					label: 'ratio',
					text: {
						ratio: statusText,
						active: statusText,
						success: statusText
					},
					autoSuccess: false
				});

				$(v.voteElm).progress({
					value: v.votes,
					total: tVotes,
					autoSuccess: false
				});

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
