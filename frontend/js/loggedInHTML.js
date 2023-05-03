import { sendMessage } from "./chat.js";

export function createLoggedInHTML() {
    const mainDiv = document.getElementById("main");
    mainDiv.innerHTML = `
    <!-- Navgation header -->
    <header class="header">
      <h1 id="title"><a>theDialectic</a></h1>
      <button type="submit" class="header-btns" id="all-posts-button">Posts</button>
      <button type="submit" class="header-btns" id="create-post-button">New Post</button>
      <button type="submit" class="header-btns" id="logout-button">Logout</button>
    </header>

    <div id="msg">
    </div>

    <!--Introductory remarks-->
    <div class="intro" id="intro">
      <h2>Welcome to theDialectic</h2>
      <p>
        Please feel free to bombard us with your conversation.<br><br>
        If you are not yet a member, please oblige us and register.<br>
        If you already have an account, login and converse.<br><br>
      </p>
    </div>

    <!-- Main Content -->
    <div class="main-content" id="main-content">
          <h3>Latest Posts</h3>
          <table id="posts-table">
            <tr>
              <th>Title</th>
              <th>Content</th>
              <th>Author</th>
              <th>Created</th>
            </tr>
          </table>
    </div>

    <!-- Chat -->
    <div class="chat" id ="chat">
      <h3>Chat</h3> 
          <textarea name="chat-messages" id="chat-messages" cols="30" rows="10" style="resize:none"></textarea>
      <div class="chat-footer">
        <!-- If using a from tag in an SPA (single page application) you must suppress the form reloading the whole page. -->
        <form>
          <textarea type="text" id="new-message" name="new-message" placeholder="Type your message"></textarea>
          <button id="message-submit" type="submit">Send</button>
        </form>
      </div>
    </div>
      `;

    // Add event listeners to the buttons
    // This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
    // document.getElementById('new-message').onsubmit = sendMessage;
    document.getElementById('chat').addEventListener('submit', function(event) {
      event.preventDefault();
      sendMessage();
    });
    // document.getElementById('logout').addEventListener('click', function(event) {
    //   event.preventDefault();
    //   logout();
    // });
  }