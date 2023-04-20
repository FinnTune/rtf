import { connectWebSocket } from './websocket.js';

export function login() {
    //Below are two different was to get the form data

    //Using new FormData dircetly into JSON.stringify does not work without the following two lines.
    // let loginFormData = new FormData(document.getElementById('login-form'));
    // let formDataArray = Array.from(loginFormData.entries());
    // let formDataJson = Object.fromEntries(formDataArray);

    //Using document.getElementById for specific fields
    let formData = {
        username: document.getElementById('username').value,
        password: document.getElementById('password').value,
    };

    // console.log(loginFormData);
    console.log(formData);

    fetch('login', {
        method: 'POST',
        body: JSON.stringify(formData),
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
            return response.json();
        } else {
            // throw new Error('Unauthorized');
            throw 'Unauthorized';
        }
    }).then((data) => {
        //At this point user is authenticated
        connectWebSocket(data.otp);
    }).catch((error) => {
        alert("Err: " + error);
        console.log("Err: ", error);
    });
 
    return false;
}