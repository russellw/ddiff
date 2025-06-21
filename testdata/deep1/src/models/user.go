package models

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func (u *User) IsAdult() bool {
    return u.Age >= 18
}