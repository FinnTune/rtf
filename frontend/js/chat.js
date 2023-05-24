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
        msgArea1.innerHTML += formattedMsg;
        //Intrusive for the user attempting to read prevuous messages
        //because it scrolls to the bottom of the chat area.
        msgArea1.scrollTop = msgArea1.scrollHeight;
    } else if (document.getElementById('chat-messages-' + event.to)) {
    let msgArea2 = document.getElementById('chat-messages-' + event.to);
        msgArea2.innerHTML += formattedMsg;
        msgArea2.scrollTop = msgArea2.scrollHeight;
    }else {
        console.log("Chat window not open");
        let usersList = document.getElementById('users-list');
        const msgAlert = document.createElement("span");
        msgAlert.innerHTML = "!";
        //Add msgAlert to the user's name in the users list
        for (let i = 0; i < usersList.children.length; i++) {
            if (usersList.children[i].textContent == event.from) {
                usersList.children[i].appendChild(msgAlert);
            }
        }
    }
}

function sendEvent(eventName, payload) {
    let event = new Event(eventName, payload);
    appendChatMsg(event.payload);
    conn.send(JSON.stringify(event));
}

export function sendMessage (message, user) {
    // var newmessage = document.getElementById('new-message');
    //Get usernmae from local storage
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
    let usersList = document.getElementById('users-list');
    usersList.innerHTML = "";
    // let users = JSON.parse(new TextDecoder().decode(new Uint8Array(event.payload))); // parse the JSON object
    let users = event.payload;

// Loop through the keys in the users object
for (let user in users) {
    // user will be the key (username), and users[user] will be the value (admin status)

    let newUser = document.createElement('li');
        
    // If the user is an admin, indicate that in the list
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
      openChatWindow(user);
      console.log("Clicked on user: " + user);
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
      <div name="chat-messages" id="chat-messages-${user}" class="chat-messages"></div>
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
    // Add the event listener to the close button
    chatWindow.querySelector('#close-chat').addEventListener('click', () => {
     mainDiv.removeChild(chatWindow);
    });
  
    // Append the chat window to the document body
    mainDiv.appendChild(chatWindow);
  }
  


// Function to get the conversation history between two users
// function getChatHistory(user, callback) {
//   // Send a message to the server to request the conversation history
//   const message = JSON.stringify({ type: "get_chat_history", user: user });
//   console.log("Sending message to backend!: " + message);
//   conn.send(message);

//   // Listen for the response from the server
//   conn.addEventListener("message", (event) => {
//     const message = JSON.parse(event.payload);
//     console.log("Message received from backend!: " + JSON.stringify(message));
//     if (message.type === "chat_history") {
//       const chatHistory = JSON.stringify(message);
//       const TempText = JSON.stringify(message);
//       console.log("TempText: " + TempText);
//       //for (const message of chatHistory) {
//       //  console.log("Message: " + JSON.stringify(message));
//       //}
//       callback(chatHistory);
//     }
//   });
// }
