package engine

func GetPiecesMap() map[string]string {
	piecesMap := make(map[string]string)
	piecesMap["wP"] = "static/images/wP.svg"
	piecesMap["wR"] = "static/images/wR.svg"
	piecesMap["wN"] = "static/images/wN.svg"
	piecesMap["wB"] = "static/images/wB.svg"
	piecesMap["wQ"] = "static/images/wQ.svg"
	piecesMap["wK"] = "static/images/wK.svg"
	piecesMap["bP"] = "static/images/bP.svg"
	piecesMap["bR"] = "static/images/bR.svg"
	piecesMap["bN"] = "static/images/bN.svg"
	piecesMap["bB"] = "static/images/bB.svg"
	piecesMap["bQ"] = "static/images/bQ.svg"
	piecesMap["bK"] = "static/images/bK.svg"
	piecesMap["---"] = ""

	return piecesMap
}
