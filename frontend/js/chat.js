//Messaging events and functions

//Import websocket conn from main.js
import {conn} from './websocket.js';

export class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
};

class SendMessageEvent {
    constructor(message, from, to){
        this.message = message;
        this.from = from;
        this.to = to;
        this.sent = Date.now();
    }
}

class ReceiveMessageEvent{
    constructor(message, from, to, sent){
        this.message = message;
        this.from = from;
        this.to = to;
        this.sent = sent;
    }
}

class GetChatHistoryEvent {
    constructor(from, to, offset, limit) {
        this.from = from;
        this.to = to;
        this.offset = offset;
        this.limit = limit;
    }
}

class TypingEvent {
    constructor(from, to) {
        this.from = from;
        this.to = to;
    }
}

export function routeEvent(event) {
    if (event.type ==undefined) {
        alert("No type field in the event.");
        console.log("Event type is undefined");
        return;
    }
    switch (event.type) {
        case "sent-message":
            const messageEvent = Object.assign(new ReceiveMessageEvent, event.payload)
            console.log("New message: ", event.payload);
             appendChatMsg(messageEvent);
            break;
        case "users-online":
            //Functionality to display online users
            const usersOnline = event;
            appendUsers(usersOnline);
            break;
        case "chat_history":
            //Functionality to display chat history
            if (event.payload == null) {
                console.log("No more chat history");
                return;
            } else {
            console.log("Appending Chat History: ", event.payload)
            // Reverse the array
            let events = event.payload.reverse();
            // document.getElementById('chat-messages-' + event.payload.to).innerHTML = "";
            events.forEach(event => {
                console.log(event)
                prependChatMsg(event);
            });
            }
            break;
        case "typing":
            if (document.getElementById('typing-indicator-' + event.payload.from)) {
                document.getElementById('typing-indicator-' + event.payload.from).style.display = 'block';
            }
            break;
        case "stop-typing":
            if (document.getElementById('typing-indicator-' + event.payload.from)) {
            document.getElementById('typing-indicator-' + event.payload.from).style.display = 'none';
            }
            break;
        case "error":
            console.log("Error: ", event.payload);
            break;
        default:
            alert("Unsupported event type: " + event.type);
            console.log("Unknown event type: ", event.type);
    }
}

function appendChatMsg(event) {
    var date = new Date(event.sent);
    const formattedMsg = `<strong>${event.from} (${date.toLocaleDateString()}-${date.toLocaleTimeString()}): </strong>${event.message.replace(/\n/g, '<br>')}<br>`;
    if (document.getElementById('chat-messages-' + event.from)) {
        let msgArea1 = document.getElementById('chat-messages-' + event.from);
        let spacer1 = msgArea1.querySelector('.spacer');
        if (spacer1) {
            spacer1.insertAdjacentHTML('beforebegin', formattedMsg);
        } else {
            msgArea1.innerHTML += formattedMsg;
        }
        msgArea1.scrollTop = msgArea1.scrollHeight;
    } else if (document.getElementById('chat-messages-' + event.to)) {
        let msgArea2 = document.getElementById('chat-messages-' + event.to);
        let spacer2 = msgArea2.querySelector('.spacer');
        if (spacer2) {
            spacer2.insertAdjacentHTML('beforebegin', formattedMsg);
        } else {
            msgArea2.innerHTML += formattedMsg;
        }
        msgArea2.scrollTop = msgArea2.scrollHeight;
    } else {
        console.log("Chat window not open");
        let usersList = document.getElementById('users-list');
        const msgAlert = document.createElement("span");
        msgAlert.className = "msg-alert";
        msgAlert.innerHTML = "!";
        let localUser = localStorage.getItem("username")
        console.log("Local User: ", localUser)
        console.log("Event.from: ", event.from)
        //Add msgAlert to the user's name in the users list
        if (localUser != event.from) {
            for (let i = 0; i < usersList.children.length; i++) {
                if (usersList.children[i].textContent == event.from) {
                    usersList.children[i].appendChild(msgAlert);
                }
            }
        }
    }
}


function prependChatMsg(event) {
    var date = new Date(event.created_at);
    console.log("Date: ", date)
    const formattedMsg = `<strong>${event.from} (${date.toLocaleDateString()}-${date.toLocaleTimeString()}): </strong>${event.message.replace(/\n/g, '<br>')}<br>`;
    let msgArea;
    if (document.getElementById('chat-messages-' + event.from)) {
        msgArea = document.getElementById('chat-messages-' + event.from);
    } else if (document.getElementById('chat-messages-' + event.to)) {
        msgArea = document.getElementById('chat-messages-' + event.to);
    } else {
        console.log("Chat window not open");
        let usersList = document.getElementById('users-list');
        const msgAlert = document.createElement("span");
        msgAlert.className = "msg-alert";
        msgAlert.innerHTML = "!";
        let localUser = localStorage.getItem("username")
        console.log("Local User: ", localUser)
        console.log("Event.from: ", event.from)
        //Add msgAlert to the user's name in the users list
        if (localUser != event.from) {
            for (let i = 0; i < usersList.children.length; i++) {
                if (usersList.children[i].textContent == event.from) {
                    usersList.children[i].appendChild(msgAlert);
                }
            }
        }
        return; // Exit if chat window is not open
    }
    msgArea.innerHTML = formattedMsg + msgArea.innerHTML; // Prepend new message
    // Save the current scroll position
    let savedScrollTop = msgArea.scrollTop;
    setTimeout(function() {
        msgArea.scrollTop = savedScrollTop;
    }, 0);
}



function sendEvent(eventName, payload) {
    let event = new Event(eventName, payload);
    appendChatMsg(event.payload);
    conn.send(JSON.stringify(event));
}

export function sendMessage (message, user) {
    //Get usernmae from local storage and wrap mesage details in SendMessageEvent
    let username = localStorage.getItem('username');
    if(message != null) {
        //Hard-coded value of the username needs to be changed???
        let outGoingMsg = new SendMessageEvent(message, username, user);
        sendEvent("new-message", outGoingMsg);
        console.log("New Message Print: ", message);
    }
    return false
}

export function appendUsers(event) {
    console.log("Users: ", event.payload)
    let currUser = localStorage.getItem("username")
    let usersList = document.getElementById('users-list');
    usersList.innerHTML = "";
    // let users = JSON.parse(new TextDecoder().decode(new Uint8Array(event.payload))); // parse the JSON object
    let users = event.payload;

    // Loop through the keys in the users object and add green cicrlce to indicate online
    for (let user in users) {

    let newUser = document.createElement('li');
        
    newUser.textContent = user;
    
    const greenCircle = document.createElement("span");
    greenCircle.style.backgroundColor = "green";
    greenCircle.style.width = "10px";
    greenCircle.style.height = "10px";
    greenCircle.style.borderRadius = "50%";
    greenCircle.style.display = "inline-block";
    greenCircle.style.marginRight = "10px";
    
    newUser.appendChild(greenCircle);

    newUser.addEventListener("click", () => {
        // New code: remove "!" sign when the user is clicked
        const msgAlerts = newUser.getElementsByClassName('msg-alert');
        for (let i = 0; i < msgAlerts.length; i++) {
            msgAlerts[i].remove();
        }

        if (user != currUser) {
            openChatWindow(user);
            console.log("Clicked on user: " + user);
        } else {
            console.log("This is you foo!")
        };
    });
    
    usersList.appendChild(newUser);
}
}


// Function to open a chat window between two users
function openChatWindow(user) {
    let mainDiv = document.getElementById('main');
    // Create a new chat window
    let chatWindow = document.createElement('div');
    chatWindow.id = 'chat:' + user;
    chatWindow.classList.add('chat-window');
  
    // Add the inner HTML content to the chat window
    chatWindow.innerHTML = `
      <h3>Chat with ${user}</h3>
      <button id="close-chat" class="close-chat">x</button>
      <div name="chat-messages" id="chat-messages-${user}" class="chat-messages" style="overflow-y: scroll;">
      <div class="spacer" style="height: 20px;"></div>
      </div>
      <div class="typing">
      <img id="typing-indicator-${user}" src="/../img/typing.gif" style="display: none; width: 30px; height: 30px;">
      </div><br>
      <div class="chat-footer">
        <form>
          <textarea type="text" id="new-message-${user}" name="new-message" placeholder="Type your message"></textarea>
          <button id="message-submit-${user}" class="btns" type="submit">Send</button>
        </form>
      </div>
    `;

    let messageSubmitButton = chatWindow.querySelector('#message-submit-' + user);
    let newMessageInput = chatWindow.querySelector('#new-message-'+ user);

    // Add event listener to the send button
    messageSubmitButton.addEventListener('click', (e) => {
        e.preventDefault(); // to prevent form submission

        // Get message text
        let messageText = newMessageInput.value;
        
        // Create a message object
        sendMessage(messageText, user);
        
        // Clear message text area
        newMessageInput.value = '';
    });

    let typingTimeout;
    
    newMessageInput.addEventListener('input', () => {
        // User started typing
        clearTimeout(typingTimeout);

        const typingEvent = new Event('typing', new TypingEvent(localUserId, user));
        conn.send(JSON.stringify(typingEvent));
        
        // User stopped typing after 1 second
        typingTimeout = setTimeout(() => {
            const stopTypingEvent = new Event('stop-typing', new TypingEvent(localUserId, user));
            conn.send(JSON.stringify(stopTypingEvent));
        }, 1000);
    });

    let offset = 0;
    const limit = 10;

    // Add the event listener to the close button
    chatWindow.querySelector('#close-chat').addEventListener('click', () => {
     mainDiv.removeChild(chatWindow);
    });
  
    // Append the chat window to the document body
    mainDiv.appendChild(chatWindow);
    // Now that the chat window is open, we can load the chat history.
    console.log("Getting chat history")
    const localUserId = localStorage.getItem("username");  // Assuming you save user ID in localStorage
    console.log("History from: ", localUserId)
    console.log("History to: ", user) 
    
    const getChatHistoryEvent = new Event("get-chat-history", new GetChatHistoryEvent(localUserId, user, offset, limit));
    conn.send(JSON.stringify(getChatHistoryEvent));


    //Get more chat history on scroll
    let chatMessagesDiv = chatWindow.querySelector('#chat-messages-' + user);

    chatMessagesDiv.addEventListener('scroll', () => {
    if (chatMessagesDiv.scrollTop === 0) {
        // The user has scrolled to the top of the chat window, so load more chat history.
        offset += limit;
        const getMoreChatHistoryEvent = new Event('get-more-chat-history', new GetChatHistoryEvent(localUserId, user, offset, limit));
        conn.send(JSON.stringify(getMoreChatHistoryEvent));
    }
    });
}
