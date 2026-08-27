import { createPostsTable, getAllPosts, showEmptyState, renderPostRows } from "./getAllPosts.js";
import { clearTable } from "./getAllPosts.js";
import { showMessage } from "./notify.js";
import { getCategories } from "./categories.js";

export async function createCategoryFilter() {
    const categoryFilterDiv = document.getElementById('category-selection');
    categoryFilterDiv.className = "category-filter";
    categoryFilterDiv.replaceChildren();
    const heading = document.createElement('h4');
    heading.textContent = 'Filter by Category:';
    categoryFilterDiv.appendChild(heading);

    let categories;
    try {
        categories = await getCategories();
    } catch (error) {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
        return categoryFilterDiv;
    }

    for (let category of categories) {
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.name = 'category';
        checkbox.value = category.name;
        checkbox.id = 'filter-category-' + category.id;

        const label = document.createElement('label')
        label.htmlFor = checkbox.id;
        label.appendChild(document.createTextNode(category.name));

         // Create a new div
         const div = document.createElement('div');

         // Append checkbox and label to the div
         div.appendChild(checkbox);
         div.appendChild(label);

        categoryFilterDiv.appendChild(div);

        // Add event listener
        checkbox.addEventListener('change', getPostsByCategory);
    }

    return categoryFilterDiv;  // Line break for readability
}

export function getPostsByCategory() {
    document.getElementById('main-content').innerHTML = "";
    console.log("Getting posts by category.")
     // Collect all the selected categories
     let selectedCategories = Array.from(document.querySelectorAll('input[type="checkbox"]:checked')).map(checkbox => checkbox.value);
    console.log("Selected categories: ", selectedCategories);
     // If no categories are selected, get all posts instead
     if (selectedCategories.length == 0) {
         return getAllPosts();
     }
 
     fetch('getPostsByCategory', {
         method: 'POST',
         mode: 'cors',
         headers: {
             'Content-Type': 'application/json'
         },
         body: JSON.stringify({ categories: selectedCategories }) // Send the selected categories to the server
     }).then((response) => {
        if(response.ok){
            console.log("Received posts by category.")
            if (document.getElementById('posts')) {
                clearTable();
            } else {
                createPostsTable();
            }
            let posts = response.json();
            console.log("PostsBef:", posts);
            return posts;
        }
        return response.text().then((message) => {
            throw new Error(message || `Failed to load posts (${response.status})`);
        });
    }).then((posts) => {
        console.log("PostsAft: ", posts)
        // Posts already arrive newest-first from the server.
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
        if (posts.length == 0) {
            showEmptyState(tbody, "No posts for this category.");
            return
        }
        renderPostRows(tbody, posts);
        return
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
        return
    });
    return
}