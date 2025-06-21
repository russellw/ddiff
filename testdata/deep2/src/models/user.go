package models

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Age   int    `json:"age"`
    Email string `json:"email"`
}

func (u *User) IsAdult() bool {
    return u.Age >= 21
}

func (u *User) GetDisplayName() string {
    return u.Name
}