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

// Countdown to page refresh
window.onload = function exampleFunction() {
	var timer = document.querySelector("body").dataset.timer;
	var lastRefresh = 0;
	function myLoop() {
		setTimeout(function() {

		document.getElementById("my-timer").innerHTML = lastRefresh + 's';
		lastRefresh += 1;
		myLoop();
		  
		}, 1000)
	  }
	  myLoop();
}
