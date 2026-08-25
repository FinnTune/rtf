import { createMainHTML } from "./mainHTML.js";
import { showMessage, setButtonLoading } from "./notify.js";

export function register() {
     //Check if password and confirm password are the same
     let password = document.getElementById('regpassword').value;
     console.log(password);
     let confirmPassword = document.getElementById('regconfpassword').value;
     console.log(confirmPassword);
     if(password != confirmPassword){
         showMessage("Passwords do not match", "error");
         return false;
     }
    //Using document.getElementById for specific fields
    let formData = {
        fname: document.getElementById('regfname').value,
        lname: document.getElementById('reglname').value,
        uname: document.getElementById('reguname').value,
        email: document.getElementById('regemail').value,
        age: document.getElementById('regage').value,
        gender: document.getElementById('reggender').value,
        password: document.getElementById('regpassword').value,
    };
    // if(!fname && !lname){
    //     return
    // } For preventing null exceptions

    // console.log(loginFormData);
    console.log(formData);

    const submitButton = document.getElementById('register-submit-button');
    setButtonLoading(submitButton, true, 'Registering...');

    fetch('register', {
        method: 'POST',
        body: JSON.stringify(formData),
        mode: 'cors', // not needed
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
        console.log("User registered.")
            createMainHTML();
            showMessage('You are now registered. Please login.', 'success');
            return;
        }
        return response.text().then((message) => {
            throw new Error(message || `Registration failed (${response.status})`);
        });
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
    }).finally(() => {
        setButtonLoading(submitButton, false);
    });

    return false;
}