export function getUsers() {
    console.log("Getting users...")
    fetch('/getUsers', {
        method: 'GET',
        mode: 'cors',
        headers: {
          'Content-Type': 'application/json'
        },
      //   body: JSON.stringify({ sessionID })
      })
      .then(response => response.json())
      .then(data => {
        console.log("Appending users: ", data)
        appendUsers(data);
      })
      .catch(error => {
        console.error('Error checking login status:', error);
      });
}