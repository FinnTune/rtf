//Import conn from main.js
import {conn} from './main.js';

export function sendMessage (message) {
    var newmessage = document.getElementById('new-message');
    if(newmessage != null) {
        conn.send(newmessage.value);
        console.log(newmessage);
    }
    newmessage.value = "";
    return false
}