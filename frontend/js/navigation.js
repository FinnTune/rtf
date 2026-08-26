import { getAllPosts } from "./getAllPosts.js";
import { createCategoryFilter } from "./categoryFilter.js";

// Switches the logged-in view to the (paginated, unfiltered) posts list.
// Shared by the "Posts" nav button, the site title, the "Clear search"
// button, and a successful post submission, so they all reset to the same
// consistent state.
export function showPostsView() {
    document.getElementById('msg').textContent = "";
    if (document.getElementById('single-post')) {
        document.getElementById('single-post').style.display = "none";
    }
    if (document.getElementById('add-post')) {
        document.getElementById('add-post').style.display = "none";
    }
    document.getElementById('category-selection').style.display = "flex";
    document.getElementById('main-content').style.display = "flex";
    document.getElementById('search-input').value = "";
    document.getElementById('clear-search-button').style.display = "none";
    createCategoryFilter();
    getAllPosts();
}
