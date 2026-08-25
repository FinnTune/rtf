import { createMainHTML } from "./mainHTML.js";
import { showMessage } from "./notify.js";

export async function logout () {
    console.log("Logging out...")
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
          console.log("User is logged out.")
          console.log(data.loggedIn)
          
          // User is logged in
          createMainHTML();
          showMessage("You've been logged out.", 'success');
        } else {
          console.log("User logout failed.")
          console.log(data.loggedIn)
          // User is not logged in
          createMainHTML();
        }
      })
      .catch(error => {
        console.error('Error logging out:', error);
        showMessage('Failed to log out. Please try again.', 'error');
      });
}