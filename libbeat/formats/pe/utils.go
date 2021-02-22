package pe

func countValue(group map[string]int, value string) {
	if found, ok := group[value]; ok {
		group[value] = found + 1
		return
	}
	group[value] = 1
}
