package api

func (rt *router) registerRoutes() {
	rt.mux.HandleFunc("/session", rt.doLogin)
	rt.mux.HandleFunc("/users", rt.users)
	rt.mux.HandleFunc("/users/me/name", rt.myUserName)
	rt.mux.HandleFunc("/users/me/photo", rt.myPhoto)
	rt.mux.HandleFunc("/conversations", rt.conversations)
	rt.mux.HandleFunc("/conversations/", rt.conversationByID)
	rt.mux.HandleFunc("/messages/", rt.messageByID)
	rt.mux.HandleFunc("/groups", rt.groups)
	rt.mux.HandleFunc("/groups/", rt.groupByID)
	rt.mux.HandleFunc("/liveness", rt.liveness)
}
