package communication

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/scribble-rs/scribble.rs/game"
	"github.com/scribble-rs/scribble.rs/state"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type UserPageData struct {
	*BasePageConfig
	WFHomieUserName string
	WFHomieToken    string
}

func createDefaultUserPageData(token string) *UserPageData {
	return &UserPageData{
		BasePageConfig:  CurrentBasePageConfig,
		WFHomieUserName: "",
		WFHomieToken:    token,
	}
}

//This file contains the API for the official web client.

// homePage servers the default page for scribble.rs, which is the page to
// create a new lobby.
func homePage(w http.ResponseWriter, r *http.Request) {
	tokens, ok := r.URL.Query()["token"]
	if !ok || len(tokens[0]) < 1 {
		err := pageTemplates.ExecuteTemplate(w, "lobby-create-page", createDefaultLobbyCreatePageData())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		token := tokens[0]
		err := pageTemplates.ExecuteTemplate(w, "enter-user-page", createDefaultUserPageData(token))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println(err)
		}
	}
}

type WFHomieResoinseApi struct {
	Session_Id string `json:"session_id"`
	Group_Id   string `json:"group_id"`
	Group_Name string `json:"group_name"`
}

//adding a new service called ssrCheckCode
func ssrCheckCode(w http.ResponseWriter, r *http.Request) {
	WFHomiecode := r.FormValue("token")
	WFHomieusername := r.FormValue("username")

	resp, err := http.Get("https://us-central1-wfhomie-85a56.cloudfunctions.net/validate?token=" + WFHomiecode)
	if err != nil {
		///handle the error on the way of calling Api here

	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//handle the error in the response of Api here

	}
	//Convert the body to type WFHomieResponseApi
	var Response WFHomieResoinseApi
	err = json.Unmarshal(body, &Response)
	if err != nil {

	}
	Response.Group_Name = "WFHomie"
	Response.Group_Id = "1"
	if Response.Group_Id+Response.Group_Name != "" {
		var lobbycheck bool = LobbyCheck(Response.Group_Id + Response.Group_Name)
		if lobbycheck == false {
			var playerName = getPlayername(r)
			LobbyCreate(playerName, Response.Group_Id+Response.Group_Name, Response.Group_Name, r, w)
		}
		http.Redirect(w, r, CurrentBasePageConfig.RootPath+"/ssrEnterLobby?lobby_id="+Response.Group_Id+Response.Group_Name+"&username="+WFHomieusername, http.StatusFound)
	} else {
		userFacingError(w, errors.New("Invalid Code!").Error())
	}
}

func LobbyCheck(value string) bool {
	lobbies := state.GetPublicLobbies()
	if lobbies != nil {
		for _, lobby := range lobbies {
			if lobby.LobbyID == value {
				return true
			}
		}
		return false
	}
	return false

}

func LobbyCreate(playername string, groupid string, groupname string, Ip *http.Request, Res http.ResponseWriter) {
	player, lobby, createError := game.CreateLobby(playername, groupid, groupname, "english", true, 75, 4, 12, 50, 3, nil, false)
	if createError != nil {
	}
	player.SetLastKnownAddress(getIPAddressFromRequest(Ip))

	http.SetCookie(Res, &http.Cookie{
		Name:     "usersession",
		Value:    player.GetUserSession(),
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	state.AddLobby(lobby)

	http.Redirect(Res, Ip, CurrentBasePageConfig.RootPath+"/ssrEnterLobby?lobby_id="+lobby.LobbyID+"&username="+playername, http.StatusFound)
}

func createDefaultLobbyCreatePageData() *CreatePageData {
	return &CreatePageData{
		BasePageConfig:    CurrentBasePageConfig,
		SettingBounds:     game.LobbySettingBounds,
		Languages:         game.SupportedLanguages,
		Public:            "true",
		DrawingTime:       "75",
		Rounds:            "4",
		MaxPlayers:        "12",
		CustomWordsChance: "50",
		ClientsPerIPLimit: "3",
		EnableVotekick:    "true",
		Language:          "english",
	}
}

// CreatePageData defines all non-static data for the lobby create page.
type CreatePageData struct {
	*BasePageConfig
	*game.SettingBounds
	Errors            []string
	WFHomiePlayerName string
	Languages         map[string]string
	Public            string
	DrawingTime       string
	Rounds            string
	MaxPlayers        string
	CustomWords       string
	CustomWordsChance string
	ClientsPerIPLimit string
	EnableVotekick    string
	Language          string
}

// ssrCreateLobby allows creating a lobby, optionally returning errors that
// occurred during creation.
func ssrCreateLobby(w http.ResponseWriter, r *http.Request) {
	formParseError := r.ParseForm()
	if formParseError != nil {
		http.Error(w, formParseError.Error(), http.StatusBadRequest)
		return
	}

	WFHomiegroupId := parseWFHomieGroupId(r.Form.Get("WFHomie_group_id"))
	// WFHomieplayerId := parseWFHomiePlayerId(r.Form.Get("WFHomie_player_id"))
	WFHomiegroupName := parseWFHomieGroupName(r.Form.Get("WFHomie_group_name"))
	// WFHmoieplayerName := parseWFHomiePlayerName(r.Form.Get("WFHomie_player_name"))
	language, languageInvalid := parseLanguage(r.Form.Get("language"))
	drawingTime, drawingTimeInvalid := parseDrawingTime(r.Form.Get("drawing_time"))
	rounds, roundsInvalid := parseRounds(r.Form.Get("rounds"))
	maxPlayers, maxPlayersInvalid := parseMaxPlayers(r.Form.Get("max_players"))
	customWords, customWordsInvalid := parseCustomWords(r.Form.Get("custom_words"))
	customWordChance, customWordChanceInvalid := parseCustomWordsChance(r.Form.Get("custom_words_chance"))
	clientsPerIPLimit, clientsPerIPLimitInvalid := parseClientsPerIPLimit(r.Form.Get("clients_per_ip_limit"))
	enableVotekick, enableVotekickInvalid := parseBoolean("enable votekick", r.Form.Get("enable_votekick"))
	publicLobby, publicLobbyInvalid := parseBoolean("public", r.Form.Get("public"))

	//Prevent resetting the form, since that would be annoying as hell.
	pageData := CreatePageData{
		SettingBounds:     game.LobbySettingBounds,
		Languages:         game.SupportedLanguages,
		Public:            r.Form.Get("public"),
		DrawingTime:       r.Form.Get("drawing_time"),
		Rounds:            r.Form.Get("rounds"),
		MaxPlayers:        r.Form.Get("max_players"),
		CustomWords:       r.Form.Get("custom_words"),
		CustomWordsChance: r.Form.Get("custom_words_chance"),
		ClientsPerIPLimit: r.Form.Get("clients_per_ip_limit"),
		EnableVotekick:    r.Form.Get("enable_votekick"),
		Language:          r.Form.Get("language"),
	}

	if languageInvalid != nil {
		pageData.Errors = append(pageData.Errors, languageInvalid.Error())
	}
	if drawingTimeInvalid != nil {
		pageData.Errors = append(pageData.Errors, drawingTimeInvalid.Error())
	}
	if roundsInvalid != nil {
		pageData.Errors = append(pageData.Errors, roundsInvalid.Error())
	}
	if maxPlayersInvalid != nil {
		pageData.Errors = append(pageData.Errors, maxPlayersInvalid.Error())
	}
	if customWordsInvalid != nil {
		pageData.Errors = append(pageData.Errors, customWordsInvalid.Error())
	}
	if customWordChanceInvalid != nil {
		pageData.Errors = append(pageData.Errors, customWordChanceInvalid.Error())
	}
	if clientsPerIPLimitInvalid != nil {
		pageData.Errors = append(pageData.Errors, clientsPerIPLimitInvalid.Error())
	}
	if enableVotekickInvalid != nil {
		pageData.Errors = append(pageData.Errors, enableVotekickInvalid.Error())
	}
	if publicLobbyInvalid != nil {
		pageData.Errors = append(pageData.Errors, publicLobbyInvalid.Error())
	}

	if len(pageData.Errors) != 0 {
		err := pageTemplates.ExecuteTemplate(w, "lobby-create-page", pageData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	var playerName = getPlayername(r)
	// var playerName = parseWFHomiePlayerName(r.Form.Get("WFHomie_player_name"))
	// var groupName = parseWFHomieGroupName(r.Form.Get("WFHomie_group_name"))

	player, lobby, createError := game.CreateLobby(playerName, WFHomiegroupId, WFHomiegroupName, language, publicLobby, drawingTime, rounds, maxPlayers, customWordChance, clientsPerIPLimit, customWords, enableVotekick)
	if createError != nil {
		pageData.Errors = append(pageData.Errors, createError.Error())
		templateError := pageTemplates.ExecuteTemplate(w, "lobby-create-page", pageData)
		if templateError != nil {
			userFacingError(w, templateError.Error())
		}
		return
	}

	player.SetLastKnownAddress(getIPAddressFromRequest(r))

	// Use the players generated usersession and pass it as a cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "usersession",
		Value:    player.GetUserSession(),
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	//We only add the lobby if we could do all necessary pre-steps successfully.
	state.AddLobby(lobby)

	http.Redirect(w, r, CurrentBasePageConfig.RootPath+"/ssrEnterLobby?lobby_id="+lobby.LobbyID, http.StatusFound)
}

func parsePlayerName(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trimmed, errors.New("the player name must not be empty")
	}

	return trimmed, nil
}

func parsePassword(value string) (string, error) {
	return value, nil
}

// func parseWFHomiePlayerName(value string) string {
// 	return value
// }
func parseWFHomieGroupName(value string) string {
	return value
}
func parseWFHomieGroupId(value string) string {
	return value
}

// func parseWFHomiePlayerId(value string) string {
// 	return value
// }

func parseLanguage(value string) (string, error) {
	toLower := strings.ToLower(strings.TrimSpace(value))
	for languageKey := range game.SupportedLanguages {
		if toLower == languageKey {
			return languageKey, nil
		}
	}

	return "", errors.New("the given language doesn't match any supported language")
}

func parseDrawingTime(value string) (int, error) {
	result, parseErr := strconv.ParseInt(value, 10, 64)
	if parseErr != nil {
		return 0, errors.New("the drawing time must be numeric")
	}

	if result < game.LobbySettingBounds.MinDrawingTime {
		return 0, fmt.Errorf("drawing time must not be smaller than %d", game.LobbySettingBounds.MinDrawingTime)
	}

	if result > game.LobbySettingBounds.MaxDrawingTime {
		return 0, fmt.Errorf("drawing time must not be greater than %d", game.LobbySettingBounds.MaxDrawingTime)
	}

	return int(result), nil
}

func parseRounds(value string) (int, error) {
	result, parseErr := strconv.ParseInt(value, 10, 64)
	if parseErr != nil {
		return 0, errors.New("the rounds amount must be numeric")
	}

	if result < game.LobbySettingBounds.MinRounds {
		return 0, fmt.Errorf("rounds must not be smaller than %d", game.LobbySettingBounds.MinRounds)
	}

	if result > game.LobbySettingBounds.MaxRounds {
		return 0, fmt.Errorf("rounds must not be greater than %d", game.LobbySettingBounds.MaxRounds)
	}

	return int(result), nil
}

func parseMaxPlayers(value string) (int, error) {
	result, parseErr := strconv.ParseInt(value, 10, 64)
	if parseErr != nil {
		return 0, errors.New("the max players amount must be numeric")
	}

	if result < game.LobbySettingBounds.MinMaxPlayers {
		return 0, fmt.Errorf("maximum players must not be smaller than %d", game.LobbySettingBounds.MinMaxPlayers)
	}

	if result > game.LobbySettingBounds.MaxMaxPlayers {
		return 0, fmt.Errorf("maximum players must not be greater than %d", game.LobbySettingBounds.MaxMaxPlayers)
	}

	return int(result), nil
}

func parseCustomWords(value string) ([]string, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil, nil
	}

	result := strings.Split(trimmedValue, ",")
	for index, item := range result {
		cases.Lower(language.English)
		trimmedItem := strings.ToLower(strings.TrimSpace(item))
		if trimmedItem == "" {
			return nil, errors.New("custom words must not be empty")
		}
		result[index] = trimmedItem
	}

	return result, nil
}

func parseClientsPerIPLimit(value string) (int, error) {
	result, parseErr := strconv.ParseInt(value, 10, 64)
	if parseErr != nil {
		return 0, errors.New("the clients per IP limit must be numeric")
	}

	if result < game.LobbySettingBounds.MinClientsPerIPLimit {
		return 0, fmt.Errorf("the clients per IP limit must not be lower than %d", game.LobbySettingBounds.MinClientsPerIPLimit)
	}

	if result > game.LobbySettingBounds.MaxClientsPerIPLimit {
		return 0, fmt.Errorf("the clients per IP limit must not be higher than %d", game.LobbySettingBounds.MaxClientsPerIPLimit)
	}

	return int(result), nil
}

func parseCustomWordsChance(value string) (int, error) {
	result, parseErr := strconv.ParseInt(value, 10, 64)
	if parseErr != nil {
		return 0, errors.New("the custom word chance must be numeric")
	}

	if result < 0 {
		return 0, errors.New("custom word chance must not be lower than 0")
	}

	if result > 100 {
		return 0, errors.New("custom word chance must not be higher than 100")
	}

	return int(result), nil
}

func parseBoolean(valueName string, value string) (bool, error) {
	if strings.EqualFold(value, "true") {
		return true, nil
	}

	if strings.EqualFold(value, "false") {
		return false, nil
	}

	if value == "" {
		return false, nil
	}

	return false, fmt.Errorf("the %s value must be a boolean value ('true' or 'false)", valueName)
}
