function toggleMobileMenu(menu) {
  menu.classList.toggle("open");
}

function applyLoader(){
	var loaderBody = `<div class="loader-container">
<div class="loader" scale=1></div>
<div>`

	// Select everything that isn't the navigation bar and remove it.
    var divs = document.querySelectorAll("body > *:not(nav)");
    for (i = 0; i < divs.length; i++) {
      divs[i].remove();
    }
    // Add the loader div the body of the HTML that's left.
    document.body.innerHTML += loaderBody;
}

// Replace the whole page body but the tabs with the loading animation.
document.addEventListener('click', function (event) {
  
  if (["A", "BUTTON"].includes(event.target.tagName)) {
	
	// Skip middle clicks and ctrl+clicks	
	if (event.ctrlKey || event.which == 2) return;
	
	// Show loading animation when clicking on links in navigation	
	var navs = document.getElementsByTagName("nav")
	console.log(navs)
	if (navs != null){
		var nav = navs[0]
		console.log(nav)
		if ( nav != null ){
			if (nav.contains(event.target)){
				applyLoader()
			}
		}
	}
	
	// Show loading animation when cliking on resource/task links
	var first_parent = event.target.parentElement
	console.log(first_parent.tagName)
	if(first_parent != null){
		if(first_parent.tagName == "P"){
			var second_parent = first_parent.parentElement
			if(second_parent != null){
				if(second_parent.tagName == "DIV"){
					var third_parent = second_parent.parentElement
					if(third_parent.tagName == "DETAILS"){
						applyLoader()
					}
				}
			}
		}
	}
  }
}, false);
