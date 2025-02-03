let timer
const tempo = 300; //Time 1000ms = 1s
let startTime;

const mouseDown = (event) => {
    startTime = Date.now();
    timer = setTimeout(() => {
        if (event.button === 0) { // Ensure it's a main click (left click)
            copyUrl(event);
        }
    }, tempo);
}

const mouseUp = (event) => {
    const duration = Date.now() - startTime;

    if (duration < tempo && event.button === 0) {
        openUrl(event);
    }
    clearTimeout(timer);
};

function openUrl(event) {
    const node = event.target.querySelector("#url");

    window.open(node.textContent, "_blank");
}

function copyUrl(event) {
    const node = event.target.querySelector("#url");
    if (node) {
        navigator.clipboard.writeText(node.textContent);
        alert("Copied: " + node.textContent);
    } else {
        alert("URL element not found.");
    }
}