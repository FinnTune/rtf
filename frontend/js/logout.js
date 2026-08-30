import { createMainHTML } from "./mainHTML.js";
import { showMessage } from "./notify.js";
import { disconnectWebSocket } from "./websocket.js";

export async function logout () {
    disconnectWebSocket();
    fetch('/logout', {
        method: 'POST',
        mode: 'cors',
        headers: {
          'Content-Type': 'application/json'
        },
      //   body: JSON.stringify({ sessionID })
      })
      .then(response => response.json())
      .then(data => {
        if (data.loggedIn == false) {
          // User is logged in
          createMainHTML();
          showMessage("You've been logged out.", 'success');
        } else {
          // User is not logged in
          createMainHTML();
        }
      })
      .catch(error => {
        console.error('Error logging out:', error);
        showMessage('Failed to log out. Please try again.', 'error');
      });
}