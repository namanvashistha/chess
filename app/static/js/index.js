
function formatUserName(name) {
    if (!name) {
        return "Anonymous";
    }
    return name.split("-").map(word => word.charAt(0).toUpperCase() + word.slice(1)).slice(0, 2).join(" ");
}
async function fetchUserToken() {
    let userToken = localStorage.getItem('userToken');
    let userData = localStorage.getItem('userData');
    let userName = "";

    try {
        if (!userToken || !userData) {
            const response = await fetch('/api/user', { method: 'POST' });
            const data = await response.json();
            if (data.response_key === "SUCCESS") {
                userToken = data.data.token;
                userName = data.data.name;
                localStorage.setItem('userToken', userToken);
                localStorage.setItem('userData', JSON.stringify(data.data));
            } else {
                console.error('Error fetching user token:', data);
                return;
            }
        } else {
            const user = JSON.parse(userData);
            userName = user.name;
        }

        // Update the DOM after data is fetched
        const userNameElement = document.getElementById('user-name');
        if (userNameElement && userName) {
            userNameElement.textContent = formatUserName(userName);
        }
    } catch (error) {
        console.error('Error fetching user token:', error);
    }

    return userToken;
}

// Call the function to fetch the user token when the page is loaded
fetchUserToken();