window.onload = function () {
    document.getElementById('new-message').onsubmit = sendmessage;
    if(window["WebSocket"]) {
        console.log("WebSocket is supported by client browser!");
        socket = new WebSocket("wss://localhost:443/ws")  //Instead of 'localhost' you can use 'document.location.host' 
    } else {
        alert("WebSocket NOT supported by client browser!");
        console.log("WebSocket NOT supported by client browser!");
    }
}