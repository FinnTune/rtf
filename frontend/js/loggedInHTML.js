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
      <div class="main-content-body">
        <div class="main-content-body-left">
          <h3>Latest Posts</h3>
          <div class="latest-posts">
            <div class="latest-post">
              <h4>Post Title</h4>
              <p>Post Content</p>
            </div>
          </div>
        </div>
        <div class="main-content-body-right">
          <h3>Latest Comments</h3>
          <div class="latest-comments">
            <div class="latest-comment">
              <h4>Comment Title</h4>
              <p>Comment Content</p>
            </div>
          </div>
        </div>
      </div>
    </div>

      <div class="chat" id ="chat">
        <div class="chat-body">
          <div class="chat-body-messages">
            <textarea name="chat-messages" id="chat-messages" cols="30" rows="10"></textarea>
          </div>
        </div>
        <div class="chat-footer">
          <!-- If using a from tag in an SPA (single page application) you must suppress the form reloading the whole page. -->
          <form>
            <input type="text" id="new-message" name="new-message" placeholder="Type your message">
            <button id="message-submit" type="submit">Send</button>
          </form>
        </div>
      </div>
      `;

    // Add event listeners to the buttons
    document.getElementById('chat').addEventListener('submit', function(event) {
      event.preventDefault();
      sendMessage();
    });
    // document.getElementById('logout').addEventListener('click', function(event) {
    //   event.preventDefault();
    //   logout();
    // });
  }