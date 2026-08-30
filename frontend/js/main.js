import { createLoggedInHTML } from './loggedInHTML.js';
import { createMainHTML } from './mainHTML.js';
import { connectWebSocket } from './websocket.js';
import { displaySinglePost, getPost } from './getAllPosts.js';
import { showMessage } from './notify.js';

window.onload = function () {
    // Check login status
    checkLoginStatus().then(loggedIn => {
      if (loggedIn) {
        openPostFromURLIfPresent();
      }
    });
};

// Supports deep-linking to /posts/:id (e.g. a bookmarked or shared post
// link) by fetching that post and opening it once the logged-in view has
// been rendered.
function openPostFromURLIfPresent() {
  const match = window.location.pathname.match(/^\/posts\/(\d+)$/);
  if (!match) {
    return;
  }
  getPost(match[1]).then((post) => {
    displaySinglePost(post);
  }).catch((error) => {
    showMessage("Err: " + error.message, "error");
    console.error(error);
  });
}

export function checkLoginStatus() {
  return fetch('/checkLogin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
  })
  .then(response => response.json())
  .then(data => {
    if (data.loggedIn) {
      createLoggedInHTML();
      connectWebSocket(data);
      return data.loggedIn;
    } else {
      createMainHTML();
      return data.loggedIn;
    }
  }).catch(error => {
    console.error('Error checking login status:', error);
    return false;
  });
}

  //ChatGPT suggestion for grabbing cookie with JS!!!
  // Helper function to get the value of a cookie
//   function getCookie(name) {
//     const value = `; ${document.cookie}`;
//     const parts = value.split(`; ${name}=`);
//     if (parts.length === 2) {
//       return parts.pop().split(';').shift();
//     }
//   }