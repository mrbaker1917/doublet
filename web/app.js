const startScreen = document.getElementById("start-screen");
const gameScreen = document.getElementById("game-screen");
const startForm = document.getElementById("start-form");
const moveForm = document.getElementById("move-form");
const startError = document.getElementById("start-error");
const moveMessage = document.getElementById("move-message");
const gameResult = document.getElementById("game-result");
const suggestionsEl = document.getElementById("suggestions");
const difficultyInput = document.getElementById("difficulty-input");
const maxLabel = document.getElementById("max-label");
const newGameBtn = document.getElementById("new-game-btn");
const solutionWrap = document.getElementById("solution-wrap");
const solutionPath = document.getElementById("solution-path");

let activeGame = null;

function showError(el, message) {
  if (!message) {
    el.hidden = true;
    el.textContent = "";
    return;
  }
  el.hidden = false;
  el.textContent = message;
}

function formatHistory(words) {
  return words.join(" → ");
}

function updateGameView(game) {
  document.getElementById("current-word").textContent = game.current;
  document.getElementById("target-word").textContent = game.end.toUpperCase();
  document.getElementById("moves-left").textContent = Math.max(game.maxChanges - game.movesUsed, 0);
  document.getElementById("history").textContent = formatHistory(game.history);
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || "Request failed");
  }
  return data;
}

async function loadSuggestions() {
  try {
    const data = await api("/api/suggestions");
    suggestionsEl.hidden = false;

    for (const button of suggestionsEl.querySelectorAll(".chip")) {
      const level = button.dataset.level;
      const pair = data[level];
      const label = level.charAt(0).toUpperCase() + level.slice(1);
      button.textContent = `${label}: ${pair[0]} → ${pair[1]}`;
      button.onclick = () => {
        document.getElementById("start-input").value = pair[0];
        document.getElementById("end-input").value = pair[1];
        difficultyInput.value = level;
        maxLabel.hidden = true;
      };
    }
  } catch {
    suggestionsEl.hidden = true;
  }
}

function showStartScreen() {
  activeGame = null;
  startScreen.hidden = false;
  gameScreen.hidden = true;
  showError(startError, "");
  showError(moveMessage, "");
  showError(gameResult, "");
  solutionWrap.hidden = true;
  solutionPath.textContent = "";
  moveForm.reset();
  newGameBtn.hidden = true;
  document.getElementById("move-input").disabled = false;
  moveForm.querySelector("button").disabled = false;
  loadSuggestions();
}

function showGameScreen(game) {
  activeGame = game;
  startScreen.hidden = true;
  gameScreen.hidden = false;
  showError(startError, "");
  showError(moveMessage, "");
  showError(gameResult, "");
  solutionWrap.hidden = true;
  solutionPath.textContent = "";
  updateGameView(game);
  document.getElementById("move-input").focus();
}

function finishGame(result) {
  const { won, lost } = result;
  document.getElementById("move-input").disabled = true;
  moveForm.querySelector("button").disabled = true;
  newGameBtn.hidden = false;

  if (won) {
    gameResult.hidden = false;
    gameResult.className = "result";
    gameResult.textContent = "Congratulations! You solved the doublet.";
    return;
  }

  if (lost) {
    gameResult.hidden = false;
    gameResult.className = "result lost";
    gameResult.textContent = "No moves left. Better luck next time.";
    if (result.solutionPath?.length) {
      solutionWrap.hidden = false;
      solutionPath.textContent = formatHistory(result.solutionPath);
    }
  }
}

difficultyInput.addEventListener("change", () => {
  maxLabel.hidden = difficultyInput.value !== "custom";
});

startForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  showError(startError, "");

  const formData = new FormData(startForm);
  const payload = {
    start: formData.get("start"),
    end: formData.get("end"),
    difficulty: formData.get("difficulty"),
  };

  if (payload.difficulty === "custom") {
    payload.max = Number(formData.get("max"));
  }

  try {
    const game = await api("/api/games", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    showGameScreen(game);
  } catch (error) {
    showError(startError, error.message);
  }
});

moveForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!activeGame) {
    return;
  }

  showError(moveMessage, "");
  const formData = new FormData(moveForm);
  const word = formData.get("word");

  try {
    const result = await api(`/api/games/${activeGame.id}/move`, {
      method: "POST",
      body: JSON.stringify({ word }),
    });

    if (!result.valid) {
      showError(moveMessage, result.message);
      return;
    }

    activeGame = {
      ...activeGame,
      current: result.current,
      movesUsed: result.movesUsed,
      history: result.history,
      status: result.won ? "won" : result.lost ? "lost" : "playing",
    };
    updateGameView(activeGame);
    moveForm.reset();
    document.getElementById("move-input").focus();

    if (result.won || result.lost) {
      finishGame(result);
    }
  } catch (error) {
    showError(moveMessage, error.message);
  }
});

newGameBtn.addEventListener("click", showStartScreen);

loadSuggestions();
