function minerOnClick(name) {
	// This is about the most basic and ugly way to do this. It should be replaced by a proper modal dialog, but it works for now.
	if (confirm("Remove inactive miner " + name + " from list?")) {
		var url = "/removeminer?minerName=" + name;
		var xhr = new XMLHttpRequest();
		xhr.open("POST", url, false);
		xhr.send();
		window.location.reload();
	}
}

function hamburgerClick(ele) {
	console.log('Hamburger!!');
	var $hamburger = $("#hamburger");
	var $sidenav = $("#sidenav");
	if ($hamburger.hasClass('sidenav-hidden')) {
		$hamburger.removeClass('sidenav-hidden');
		$sidenav.animate({
			left: "0",
		  }, 300, function() {
			$('.close_hamburger').css('display', 'block');
		  });
	} else {
		$hamburger.addClass('sidenav-hidden');
		$('.close_hamburger').css('display', 'none');
		$sidenav.animate({
			left: "-50%",
		  }, 300, function() {
		  });
	}
}

// Countdown to page refresh
window.onload = function refreshCountDown() {
	var timer = document.querySelector("body").dataset.timer;
	var lastRefresh = 0;
	function refreshLoop() {
		setTimeout(function() {

		document.getElementById("my-timer").innerHTML = lastRefresh + 's';
		lastRefresh += 1;
		refreshLoop();
		  
		}, 1000)
	  }
	  refreshLoop();
}
