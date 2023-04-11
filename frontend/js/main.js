export let conn

function doOnMessage(event) {
        console.log("Event print: ", event);
        console.log("Event data print: ", event.data);
        const eventData = JSON.parse(event.data);
        const eventObj = Object.assign(new Event, eventData);
        routeEvent(eventObj);
};

window.onload = function () {
    if(window["WebSocket"]) {
        console.log("WebSocket is supported by client browser!");
        conn = new WebSocket("wss://localhost:443/ws")  //Instead of 'localhost' you can use 'document.location.host' 
        console.log("Connection print: ",conn);
        conn.onmessage = doOnMessage;
    } else {
        alert("WebSocket NOT supported by client browser!");
        console.log("WebSocket NOT supported by client browser!");
    };
};
