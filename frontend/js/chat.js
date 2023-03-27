function sendmessage (message) {
    var newmessage = document.getElementById('new-message');
    if(newmessage != null) {
        console.log(newmessage)
    }
    var data = {
        message: message,
        room: room
    };
    socket.emit('sendmessage', data);
    return false
}