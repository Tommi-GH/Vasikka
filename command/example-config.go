package your_package

func getTeamInfo(token string) teamInfo {

	example_teams := make(map[string]teamInfo)

	teams["token_team1"] = team1
	teams["token_team2"] = team2
	teams["token_team3"] = team3

	return teams[token]

}
