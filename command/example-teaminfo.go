package listener

//Teaminformation
var exampleTeam = teamInfo{
	PublicSlackurl:      "",            //webhook URL for posting messages as bot
	PrivateSlackurl:     "",            //webhook URL for posting messages to another channel as bot
	TargetSpreadsheetID: "",            //Google Sheets ID for sheet with keywords
	ReadRange:           "A2:B",        //A1-notation for the range where keywords are
	ReportSpreadsheetID: "",            //Google Sheets ID for sheet where the reports are written
	WriteRange:          "Sheet1!A2:D", //A1-notation for the range where the reports are written
	ErrorMessage:        "",            //Message that is sent as a response, if there's an error
	NoTargetMessage:     "",            //Message that is sent as a response, if no keyword is found
	Answer:              "",            //Message that is sent as a response, if everything was okay. Sender's name is added before this message
	NoPostKey:           "",            //Keyword for not posting the message to the specified channel, only saving it
	HelpText:            "",            //Helptext that is shown with the message "-help"
	AnswerPrefix:        "",            //Prefix for the Answer before sender's name: "Prefix *sender's name* Answer"
}
