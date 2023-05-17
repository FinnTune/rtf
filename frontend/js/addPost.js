export function addPost() {
  var title = document.getElementById('post-title').value;
  var content = document.getElementById('post-content').value;
  var selectedCategories = Array.from(document.querySelectorAll('input[name="categories[]"]:checked')).map(function(category) {
    return {
      id: parseInt(category.getAttribute('title')),
      name: category.getAttribute('value')
    };
  });

  const uname = localStorage.getItem('username');
  const userID = parseInt(localStorage.getItem('id'));

  var postData = {
    title: title,
    content: content,
    userID: userID,
    categories: selectedCategories,
    author: uname
  };

  console.log("PostData: ",postData)

  fetch('/addPost', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(postData)
  })
  .then(function(response) {
    if (response.ok) {
      console.log('Post data sent successfully!');
      document.getElementById('msg').innerHTML = "Your post was submitted."
      // Add any additional logic or UI updates here after successful submission
    } else {
      console.error('Error sending post data.');
      // Handle the error condition appropriately
    }
  })
  .catch(function(error) {
    console.error('Error sending post data:', error);
    // Handle the error condition appropriately
  });
};