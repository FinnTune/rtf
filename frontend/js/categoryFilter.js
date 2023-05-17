import { createPostsTable, getAllPosts } from "./getAllPosts.js";

export function createCategoryFilter() {
    const categoryFilterDiv = document.getElementById('category-selection');
    categoryFilterDiv.className = "category-filter";
    categoryFilterDiv.innerHTML = `<h4>Filter by Category:</h4>`;

    const categories = [
        "Cuisine", "Places", "Activities", "Events", "Code", 
        "Language", "Sports", "Politics", "Social", "Religion",
        "Business", "Geography", "Science", "Health", "Other"
    ];

    for (let category of categories) {
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.name = 'category';
        checkbox.value = category;
        checkbox.id = category;
        
        const label = document.createElement('label')
        label.htmlFor = category;
        label.appendChild(document.createTextNode(category));

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

export function getPostsByCategory(categoryId) {
    document.getElementById('main-content').innerHTML = "";
    console.log("Getting posts by category.")
     // Collect all the selected categories
     const selectedCategories = Array.from(document.querySelectorAll('input[type="checkbox"]:checked')).map(checkbox => checkbox.value);
    console.log("Selected categories: ", selectedCategories);
     // If no categories are selected, get all posts instead
     if (selectedCategories.length === 0) {
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
    }).then((posts) => {
        console.log("PostsAft: ", posts)
        posts.sort((a, b) => (a.CreatedAt > b.CreatedAt) ? -1 : 1);
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
        if (posts.length == 0) {
            table.innerHTML = "No posts for this category."
        }
        for(let i = 0; i < posts.length; i++){
            let row = tbody.insertRow();
            let title = row.insertCell(0);
            let content = row.insertCell(1);
            let author = row.insertCell(2);
            let dateCreated = row.insertCell(3);
            let link = document.createElement("a");
            link.href = "/posts/" + posts[i].Id;
            link.className = "post-link";
            link.innerHTML = posts[i].Title;
            link.addEventListener("click", function(event){
                event.preventDefault();
                displaySinglePost(posts[i]);
            });
            title.appendChild(link);
            content.innerHTML = posts[i].Content;
            author.innerHTML = posts[i].Author;
            dateCreated.innerHTML = posts[i].Created;
        }
        return
    }).catch((error) => {
        alert("Err: " + error);
        console.log("Err: ", error);
        return
    });
    return
}