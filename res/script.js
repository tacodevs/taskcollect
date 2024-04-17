var loaderBody = `<div class="loader-container">
<div class="loader" scale=1></div>
<div>`

function toggleMobileMenu(menu) {
  menu.classList.toggle("open");
}

function applyLoader() {
  var divs = document.querySelectorAll("body > *:not(header)");
  for (i = 0; i < divs.length; i++) {
    divs[i].remove();
  }
  document.body.innerHTML += loaderBody;
}

// Replace the whole page body but the navbar with the loading animation.
document.addEventListener("click", function (event) {
  // Skip middle clicks and ctrl+clicks
  if (event.ctrlKey || event.which == 2) {
    return;
  }

  if (["A", "BUTTON"].includes(event.target.tagName)) {
    // Show loading animation when clicking on links in navigation	
    let navs = document.getElementsByTagName("nav");
    if (navs != null) {
      let nav = navs[0];
      if (nav != null && nav.contains(event.target)) {
        applyLoader();
      }
    }

    // Show loading animation when cliking on resource/task links
    let parent = event.target.parentElement;
    if (parent != null && parent.tagName == "P") {
      let grandparent = parent.parentElement;
      if (grandparent != null && grandparent.tagName == "DIV") {
        let greatGrandparent = grandparent.parentElement;
        if (greatGrandparent.tagName == "DETAILS") {
          applyLoader();
        }
      }
    }
  }
}, false);
