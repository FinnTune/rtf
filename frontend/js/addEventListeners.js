import { logout } from "./logout.js";
import { addPost } from "./addPost.js";
import { searchPosts } from "./search.js";
import { showPostsView } from "./navigation.js";

export function addEventListeners() {
    // Add event listeners to the buttons
    // This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
    // document.getElementById('new-message').onsubmit = sendMessage;
    document.getElementById('all-posts-button').addEventListener('click', function(event) {
        event.preventDefault();
        if (document.getElementById('intro')) {
            document.getElementById('intro').style.display = "none";
        }
        showPostsView();
    });

    document.getElementById('create-post-button').addEventListener('click', function(event) {
        event.preventDefault();
        document.getElementById('add-post').style.display = "flex";
        document.getElementById('main-content').style.display = "none";
        document.getElementById('intro').style.display = "none";
        document.getElementById('category-selection').style.display = "none";
    });

    document.getElementById('logout-button').addEventListener('click', function(event) {
    event.preventDefault();
    logout();
    });

    document.getElementById('title').addEventListener('click', function() {
        if (document.getElementById('intro')) {
            document.getElementById('intro').style.display = "flex";
        }
        showPostsView();
    });

    if (document.getElementById('add-post-form')) {
        document.getElementById('add-post-form').addEventListener('submit', function(event) {
            event.preventDefault();
            addPost();
        });
    }

    document.getElementById('search-form').addEventListener('submit', function(event) {
        event.preventDefault();
        document.getElementById('msg').textContent = "";
        if (document.getElementById('single-post')) {
            document.getElementById('single-post').style.display = "none";
        }
        if (document.getElementById('intro')) {
            document.getElementById('intro').style.display = "none";
        }
        if (document.getElementById('add-post')) {
            document.getElementById('add-post').style.display = "none";
        }
        document.getElementById('category-selection').style.display = "none";
        document.getElementById('main-content').style.display = "flex";
        searchPosts(document.getElementById('search-input').value);
    });

    document.getElementById('clear-search-button').addEventListener('click', function() {
        showPostsView();
    });
}