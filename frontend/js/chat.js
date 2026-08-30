//Messaging events and functions

//Import websocket conn from main.js
import {conn} from './websocket.js';

export class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
}

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


let alertUsers = [];

function appendMessageFragment(container, sender, message, date, prepend = false) {
    const fragment = document.createDocumentFragment();
    const header = document.createElement('strong');
    header.textContent = `${sender} (${date.toLocaleDateString()}-${date.toLocaleTimeString()}): `;
    fragment.appendChild(header);
    fragment.appendChild(document.createTextNode(String(message ?? '')));
    fragment.appendChild(document.createElement('br'));
    if (prepend) {
        container.prepend(fragment);
    } else {
        container.appendChild(fragment);
    }
}

export function routeEvent(event) {
    if (event.type ==undefined) {
        console.error("No type field in the event.");
        return;
    }
    switch (event.type) {
        case "sent-message": {
            const messageEvent = Object.assign(new ReceiveMessageEvent, event.payload)
            appendChatMsg(messageEvent);
            break;
        }
        case "users-online": {
            //Functionality to display online users
            const usersOnline = event;
            appendUsers(usersOnline);
            break;
        }
        case "chat_history": {
            //Functionality to display chat history
            if (event.payload == null) {
                return;
            } else {
            // Reverse the array
            let events = event.payload.reverse();
            events.forEach(event => {
                prependChatMsg(event);
            });
            }
            break;
        }
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
            console.error(event.payload);
            break;
        default:
            console.error("Unsupported event type: ", event.type);
    }
}

function appendChatMsg(event) {
    var date = new Date(event.sent);
    if (document.getElementById('chat-messages-' + event.from)) {
        let msgArea1 = document.getElementById('chat-messages-' + event.from);
        let spacer1 = msgArea1.querySelector('.spacer');
        if (spacer1) {
            appendMessageFragment(msgArea1, event.from, event.message, date);
        } else {
            appendMessageFragment(msgArea1, event.from, event.message, date);
        }
        msgArea1.scrollTop = msgArea1.scrollHeight;
    } else if (document.getElementById('chat-messages-' + event.to)) {
        let msgArea2 = document.getElementById('chat-messages-' + event.to);
        let spacer2 = msgArea2.querySelector('.spacer');
        if (spacer2) {
            appendMessageFragment(msgArea2, event.from, event.message, date);
        } else {
            appendMessageFragment(msgArea2, event.from, event.message, date);
        }
        msgArea2.scrollTop = msgArea2.scrollHeight;
    } else {
        let usersList = document.getElementById('users-list');
        const msgAlert = document.createElement("span");
        msgAlert.className = "msg-alert";
        msgAlert.textContent = "!";
       
    let localUser = localStorage.getItem("username");

    // Add msgAlert to the user's name in the users list
    if (localUser != event.from) {
        let userItem;
        for (let i = 0; i < usersList.children.length; i++) {
            if (usersList.children[i].textContent == event.from) {
                usersList.children[i].appendChild(msgAlert);
                userItem = usersList.children[i];  // keep the reference to the user item
            }
        }

        // Move user to the top of the list
        if (userItem) {
            usersList.insertBefore(userItem, usersList.firstChild);
        }
    }
    }
}



function prependChatMsg(event) {
    var date = new Date(event.created_at);
    let msgArea;
    if (document.getElementById('chat-messages-' + event.from)) {
        msgArea = document.getElementById('chat-messages-' + event.from);
    } else if (document.getElementById('chat-messages-' + event.to)) {
        msgArea = document.getElementById('chat-messages-' + event.to);
    } else {
        let usersList = document.getElementById('users-list');
        const msgAlert = document.createElement("span");
        msgAlert.className = "msg-alert";
        msgAlert.textContent = "!";
        let localUser = localStorage.getItem("username")
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
    appendMessageFragment(msgArea, event.from, event.message, date, true);
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
    }
    return false
}

export function appendUsers(event) {
    let currUser = localStorage.getItem("username")
    let usersList = document.getElementById('users-list');
    usersList.innerHTML = "";
    // let users = JSON.parse(new TextDecoder().decode(new Uint8Array(event.payload))); // parse the JSON object
    let users = event.payload;

      // First convert the object keys to an array
      let usersArray = Object.keys(users);

      // Now sort the array
      usersArray.sort();
  
      // Remove alertUsers from usersArray
      alertUsers = alertUsers.filter(user => usersArray.includes(user));
      usersArray = usersArray.filter(user => !alertUsers.includes(user));
  
      // Then add alertUsers to the top of usersArray
      usersArray = alertUsers.concat(usersArray);
  
      // Loop through the keys in the sorted users array and add green circle to indicate online
      for (let i = 0; i < usersArray.length; i++) {
          let user = usersArray[i];


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

        if (alertUsers.includes(user)) {
            const msgAlert = document.createElement("span");
            msgAlert.className = "msg-alert";
            msgAlert.textContent = "!";
            newUser.appendChild(msgAlert);
        }

        newUser.addEventListener("click", () => {
        // New code: remove "!" sign when the user is clicked
        const msgAlerts = newUser.getElementsByClassName('msg-alert');
        for (let i = 0; i < msgAlerts.length; i++) {
            msgAlerts[i].remove();
        }

         // Remove user from alertUsers
        alertUsers = alertUsers.filter(alertUser => alertUser != user);

        if (user != currUser) {
            openChatWindow(user);
        }
    });
    
    usersList.appendChild(newUser);
}
}


// Function to open a chat window between two users
function openChatWindow(user) {
    let mainDiv = document.getElementById('main');

    // If a chat window with this user is already open, focus it instead of
    // opening a second, duplicate window.
    const existingWindow = document.getElementById('chat:' + user);
    if (existingWindow) {
        const existingInput = existingWindow.querySelector('#new-message-' + user);
        if (existingInput) {
            existingInput.focus();
        }
        return;
    }

    // Create a new chat window. Built via createElement/textContent rather
    // than an innerHTML template string, since `user` is untrusted input as
    // far as this file is concerned — it happens to be constrained
    // server-side today (registration usernames are alphanumeric/_/- only),
    // but a DOM-injection sink shouldn't rely on a distant, unrelated
    // validation rule elsewhere in the codebase to stay safe.
    let chatWindow = document.createElement('div');
    chatWindow.id = 'chat:' + user;
    chatWindow.classList.add('chat-window');

    let heading = document.createElement('h3');
    heading.textContent = 'Chat with ' + user;

    let closeButton = document.createElement('button');
    closeButton.id = 'close-chat';
    closeButton.className = 'close-chat';
    closeButton.textContent = 'x';

    let chatMessagesDiv = document.createElement('div');
    chatMessagesDiv.setAttribute('name', 'chat-messages');
    chatMessagesDiv.id = 'chat-messages-' + user;
    chatMessagesDiv.className = 'chat-messages';
    chatMessagesDiv.style.overflowY = 'scroll';
    let spacer = document.createElement('div');
    spacer.className = 'spacer';
    spacer.style.height = '20px';
    chatMessagesDiv.appendChild(spacer);

    let typingDiv = document.createElement('div');
    typingDiv.className = 'typing';
    let typingIndicator = document.createElement('img');
    typingIndicator.id = 'typing-indicator-' + user;
    typingIndicator.src = '/img/typing.gif';
    typingIndicator.style.display = 'none';
    typingIndicator.style.width = '30px';
    typingIndicator.style.height = '30px';
    typingDiv.appendChild(typingIndicator);

    let chatFooter = document.createElement('div');
    chatFooter.className = 'chat-footer';
    let messageForm = document.createElement('form');
    let newMessageInput = document.createElement('textarea');
    newMessageInput.type = 'text';
    newMessageInput.id = 'new-message-' + user;
    newMessageInput.name = 'new-message';
    newMessageInput.placeholder = 'Type your message';
    let messageSubmitButton = document.createElement('button');
    messageSubmitButton.id = 'message-submit-' + user;
    messageSubmitButton.className = 'btns';
    messageSubmitButton.type = 'submit';
    messageSubmitButton.textContent = 'Send';
    messageForm.appendChild(newMessageInput);
    messageForm.appendChild(messageSubmitButton);
    chatFooter.appendChild(messageForm);

    chatWindow.appendChild(heading);
    chatWindow.appendChild(closeButton);
    chatWindow.appendChild(chatMessagesDiv);
    chatWindow.appendChild(typingDiv);
    chatWindow.appendChild(document.createElement('br'));
    chatWindow.appendChild(chatFooter);

    // Add event listener to the send button
    messageSubmitButton.addEventListener('click', (e) => {
        e.preventDefault(); // to prevent form submission

        // Get message text
        let messageText = newMessageInput.value.trim();
        if (!messageText) {
            return;
        }

        // Create a message object
        sendMessage(messageText, user);

        // Clear message text area
        newMessageInput.value = '';
    });

    // Enter sends the message (Shift+Enter still inserts a newline), matching
    // the behavior users expect from a chat composer.
    newMessageInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            messageSubmitButton.click();
        }
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
    closeButton.addEventListener('click', () => {
     mainDiv.removeChild(chatWindow);
    });

    // Append the chat window to the document body
    mainDiv.appendChild(chatWindow);
    // Now that the chat window is open, we can load the chat history.
    const localUserId = localStorage.getItem("username");  // Assuming you save user ID in localStorage

    const getChatHistoryEvent = new Event("get-chat-history", new GetChatHistoryEvent(localUserId, user, offset, limit));
    conn.send(JSON.stringify(getChatHistoryEvent));


    // Loading older history shifts scroll position back to (or near) 0 once
    // the new messages are prepended, and browsers keep firing 'scroll'
    // events while pinned at the top. Without a guard, that resends
    // get-more-chat-history — with an ever-incrementing offset — on every
    // one of those events instead of once per deliberate scroll-to-top.
    // There's no request/response id in this WS protocol to correlate a
    // reply back to a specific request, so a short cooldown after sending is
    // used instead of waiting for one.
    let loadingHistory = false;
    chatMessagesDiv.addEventListener('scroll', () => {
    if (chatMessagesDiv.scrollTop === 0 && !loadingHistory) {
        loadingHistory = true;
        // The user has scrolled to the top of the chat window, so load more chat history.
        offset += limit;
        const getMoreChatHistoryEvent = new Event('get-more-chat-history', new GetChatHistoryEvent(localUserId, user, offset, limit));
        conn.send(JSON.stringify(getMoreChatHistoryEvent));
        setTimeout(() => { loadingHistory = false; }, 800);
    }
    });
}
