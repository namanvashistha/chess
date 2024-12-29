package engine

func GetBoardLayout() [8][8][2]string {
	board := [8][8][2]string{}
	colors := [2]string{"w", "b"} // Alternating colors: black and white
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			file := string('a' + col) // Columns labeled as 'a' to 'h'
			rank := string('8' - row) // Rows labeled as '8' to '1'
			// Determine color based on row and column
			color := colors[(row+col)%2]
			board[row][col] = [2]string{file + rank, color}
		}
	}
	return board
}
