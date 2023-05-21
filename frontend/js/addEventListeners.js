import { sendMessage } from "./chat.js";
import { logout } from "./logout.js";
import { addPostHTML } from "./addPostHTML.js";
import { getAllPosts } from "./getAllPosts.js";
import { addPost } from "./addPost.js";
import { createCategoryFilter } from "./categoryFilter.js";
import { getUsers } from "./getUsers.js";

export function addEventListeners() {
    // Add event listeners to the buttons
    // This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
    // document.getElementById('new-message').onsubmit = sendMessage;
    document.getElementById('chat').addEventListener('submit', function(event) {
        event.preventDefault();
        sendMessage();
    });

    document.getElementById('all-posts-button').addEventListener('click', function(event) {
        event.preventDefault();
        document.getElementById('msg').innerHTML = "";
        if (document.getElementById('single-post')) {
        document.getElementById('single-post').style.display = "none";
        }
        if (document.getElementById('intro')) {
        document.getElementById('intro').style.display = "none";
        }
        if (document.getElementById('add-post')) {
        document.getElementById('add-post').style.display = "none";
        }
        document.getElementById('category-selection').style.display = "flex";
        document.getElementById('main-content').style.display = "flex";
        createCategoryFilter();
        getAllPosts();
    });

    document.getElementById('create-post-button').addEventListener('click', function(event) {
        event.preventDefault();
        addPostHTML();
        getUsers();
        document.getElementById('main-content').style.display = "none"
        document.getElementById('intro').style.display = "none";
        document.getElementById('category-selection').style.display = "none";
    });

    document.getElementById('logout-button').addEventListener('click', function(event) {
    event.preventDefault();
    logout();
    });

    document.getElementById('title').addEventListener('click', function() {
        document.getElementById('msg').innerHTML = "";
        if (document.getElementById('intro')) {
            document.getElementById('intro').style.display = "flex";
        }
        if (document.getElementById('add-post')) {
            document.getElementById('add-post').style.display = "none";
        }
        document.getElementById('category-selection').style.display = "flex";
        document.getElementById('main-content').style.display = "flex";
        createCategoryFilter();
        getAllPosts();
    });

    if (document.getElementById('add-post-form')) {
        document.getElementById('add-post-form').addEventListener('submit', function(event) {
            event.preventDefault();
            addPost();
        });
    };
};