export function addPost() {
     //Check if password and confirm password are the same
     let postTitle = document.getElementById('post-title').value;
     console.log(postTitle);
     let postContent = document.getElementById('post-content').value;
     console.log(postContent);
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

    // console.log(loginFormData);
    console.log(formData);

    fetch('register', {
        method: 'POST',
        body: JSON.stringify(formData),
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
        console.log("User registered.")
            createMainHTML();
            document.getElementById('msg').innerHTML = 'You are now registered. Please login.';
            return;
        } else {
            // throw new Error('Unauthorized');
            throw 'Unauthorized';
        }
    }).catch((error) => {
        alert("Err: " + error);
        console.log("Err: ", error);
    });
 
    return false;
};