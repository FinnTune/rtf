export function createLoggedInHTML() {
    const mainDiv = document.getElementById("main");
  
    // Navgation header
    const header = document.createElement("header");
    header.classList.add("header");
  
    const title = document.createElement("h1");
    title.id = "title";
  
    const anchor = document.createElement("a");
    anchor.textContent = "theDialectic";
    title.appendChild(anchor);
    header.appendChild(title);
  
    const allPostsButton = document.createElement("button");
    allPostsButton.type = "submit";
    allPostsButton.classList.add("header-btns");
    allPostsButton.id = "all-posts-button";
    allPostsButton.style.display = "none";
    allPostsButton.textContent = "Posts";
    header.appendChild(allPostsButton);
  
    const createPostButton = document.createElement("button");
    createPostButton.type = "submit";
    createPostButton.classList.add("header-btns");
    createPostButton.id = "create-post-button";
    createPostButton.style.display = "none";
    createPostButton.textContent = "New Post";
    header.appendChild(createPostButton);
  
    const logoutButton = document.createElement("button");
    logoutButton.type = "submit";
    logoutButton.classList.add("header-btns");
    logoutButton.id = "logout-button";
    logoutButton.textContent = "Logout";
    header.appendChild(logoutButton);
  
    mainDiv.appendChild(header);
  
    // Message div
    const msgDiv = document.createElement("div");
    msgDiv.id = "msg";
    mainDiv.appendChild(msgDiv);
  
    // Introductory remarks
    const introDiv = document.createElement("div");
    introDiv.classList.add("intro");
    introDiv.id = "intro";
  
    const introTitle = document.createElement("h2");
    introTitle.textContent = "Welcome to theDialectic";
    introDiv.appendChild(introTitle);
  
    const introParagraph = document.createElement("p");
    introParagraph.innerHTML = "Please feel free to bombard us with your conversation.<br><br>If you are not yet a member, please oblige us and register.<br>If you already have an account, login and converse.<br><br>";
    introDiv.appendChild(introParagraph);
  
    mainDiv.appendChild(introDiv);
  
    // Main content
    const mainContentDiv = document.createElement("div");
    mainContentDiv.classList.add("main-content");
    mainContentDiv.id = "main-content";
    mainContentDiv.style.display = "none";
  
    const mainContentBodyDiv = document.createElement("div");
    mainContentBodyDiv.classList.add("main-content-body");
  
    const mainContentBodyLeftDiv = document.createElement("div");
    mainContentBodyLeftDiv.classList.add("main-content-body-left");
  
    const mainContentBodyLeftTitle = document.createElement("h3");
    mainContentBodyLeftTitle.textContent = "Latest Posts";
    mainContentBodyLeftDiv.appendChild(mainContentBodyLeftTitle);
  
    const latestPostsDiv = document.createElement("div");
    latestPostsDiv.classList.add("latest-posts");
  
    const latestPostDiv = document.createElement("div");
    latestPostDiv.classList.add("latest-post");
  
    const latestPostTitle = document.createElement("h4");
    latestPostTitle.textContent = "Post Title";
    latestPostDiv.appendChild(latestPostTitle);
  
    const latestPostContent = document.createElement("p");
    latestPostContent.textContent = "Post Content";
    latestPostDiv.appendChild(latestPostContent);
  
    latestPostsDiv.appendChild(latestPostDiv);
    mainContentBodyLeftDiv.appendChild(latestPostsDiv);
  
    const mainContentBodyRightDiv = document.createElement("div");
    mainContentBodyRightDiv.classList.add("main-content-body-right");
  
    const mainContentBodyRightTitle = document.createElement("h3");
  mainContentBodyRightTitle.textContent = "Latest Comments";
  mainContentBodyRightDiv.appendChild(mainContentBodyRightTitle);
  
  const latestCommentsDiv = document.createElement("div");
  latestCommentsDiv.classList.add("latest-comments");
  
  const latestCommentDiv = document.createElement("div");
  latestCommentDiv.classList.add("latest-comment");
  
  const latestCommentTitle = document.createElement("h4");
  latestCommentTitle.textContent = "Comment Title";
  latestCommentDiv.appendChild(latestCommentTitle);
  
  const latestCommentContent = document.createElement("p");
  latestCommentContent.textContent = "Comment Content";
  latestCommentDiv.appendChild(latestCommentContent);
  
  latestCommentsDiv.appendChild(latestCommentDiv);
  mainContentBodyRightDiv.appendChild(latestCommentsDiv);
  
  mainContentBodyDiv.appendChild(mainContentBodyLeftDiv);
  mainContentBodyDiv.appendChild(mainContentBodyRightDiv);
  mainContentDiv.appendChild(mainContentBodyDiv);
  
  mainDiv.appendChild(mainContentDiv);
  
  // Chat
  const chatDiv = document.createElement("div");
  chatDiv.classList.add("chat");
  chatDiv.id = "chat";
  chatDiv.style.display = "none";
  
  const chatBodyDiv = document.createElement("div");
  chatBodyDiv.classList.add("chat-body");
  
  const chatBodyMessagesDiv = document.createElement("div");
  chatBodyMessagesDiv.classList.add("chat-body-messages");
  
  const chatMessagesTextarea = document.createElement("textarea");
  chatMessagesTextarea.name = "chat-messages";
  chatMessagesTextarea.id = "chat-messages";
  chatMessagesTextarea.cols = 30;
  chatMessagesTextarea.rows = 10;
  
  chatBodyMessagesDiv.appendChild(chatMessagesTextarea);
  chatBodyDiv.appendChild(chatBodyMessagesDiv);
  chatDiv.appendChild(chatBodyDiv);
  
  const chatFooterDiv = document.createElement("div");
  chatFooterDiv.classList.add("chat-footer");
  
  const chatForm = document.createElement("form");
  
  const newMessageInput = document.createElement("input");
  newMessageInput.type = "text";
  newMessageInput.id = "new-message";
  newMessageInput.name = "new-message";
  newMessageInput.placeholder = "Type your message";
  
  const messageSubmitButton = document.createElement("button");
  messageSubmitButton.type = "submit";
  messageSubmitButton.id = "message-submit";
  messageSubmitButton.textContent = "Send";
  
  chatForm.appendChild(newMessageInput);
  chatForm.appendChild(messageSubmitButton);
  chatFooterDiv.appendChild(chatForm);
  chatDiv.appendChild(chatFooterDiv);
  
  mainDiv.appendChild(chatDiv);
  }