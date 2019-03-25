/**
 * Copyright (C) 2019, Xiongfa Li.
 * All right reserved.
 * @author xiongfa.li
 * @version V1.0
 * Description: 
 */

package oauth2

import (
    "encoding/json"
    "github.com/emicklei/go-restful"
    "net/http"
    "github.com/xfali/oauth2/defines"
    "time"
)

func ProcessGrantTypeRefreshToken(auth *OAuth2, request *restful.Request, response *restful.Response) {
    //应用程序包含它在重定向中给出的授权码
    basic := request.HeaderParameter("Authorization")

    var client_id, client_secret string
    if basic == "" {
        tmp, err := request.BodyParameter("client_id")
        if err != nil {
            response.WriteErrorString(http.StatusBadRequest, defines.PASSWORD_CREDENTIALS_HEAD_MISSING.Error()+"and"+defines.CLINET_ID_MISSING.Error())
            return
        }
        client_id = tmp

        tmp2, err := request.BodyParameter("client_secret")
        if err != nil {
            response.WriteErrorString(http.StatusBadRequest, defines.CLIENT_SECRET_MISSING.Error())
            return
        }
        client_secret = tmp2
    } else {
        var err error = nil
        client_id, client_secret, err = parseBasicInfo(basic)
        if err != nil {
            response.WriteErrorString(http.StatusBadRequest, err.Error())
        }
    }

    //check client_id and client_secret
    secret, err := auth.ClientManager.QuerySecret(client_id)
    if err != nil {
        response.WriteErrorString(http.StatusBadRequest, defines.CHECK_CLIENT_ID_ERROR.Error())
        return
    }

    if client_secret != secret {
        response.WriteErrorString(http.StatusUnauthorized, defines.CLINET_SECRET_NOT_MATCH.Error())
        return
    }

    errCode := auth.EventListener(client_id, defines.RequestRefreshTokenEvent)
    if errCode != nil {
        response.WriteError(errCode.HttpStatus, errCode)
    }

    refresh_token, err := request.BodyParameter("refresh_token")
    if err != nil {
        response.WriteErrorString(http.StatusBadRequest, defines.REFRESH_TOKEN_MISSING.Error())
        return
    }

    jwt_client_id, err := parseToken(secret, refresh_token)
    if err != nil {
        response.WriteErrorString(http.StatusBadRequest, defines.TOKEN_ERROR.Error())
        return
    }

    if client_id != jwt_client_id {
        response.WriteErrorString(http.StatusBadRequest, defines.CHECK_CLIENT_ID_ERROR.Error())
        return
    }

    client_id_saved, err := getRefreshToken(auth.DataManager, refresh_token)
    if err != nil {
        response.WriteErrorString(http.StatusBadRequest, defines.REFRESH_TOKEN_NOT_FOUND.Error())
        return
    }

    if client_id != client_id_saved {
        response.WriteErrorString(http.StatusBadRequest, defines.CHECK_CLIENT_ID_ERROR.Error())
        return
    }

    //与请求authorization code时使用的redirect_uri相同。某些资源（API）不需要此参数。
    //redirect_uri, err := request.BodyParameter("redirect_uri")
    accessToken, err := generateToken(client_id, client_secret, defines.AccessTokenExpireTime)
    if err != nil {
        response.WriteErrorString(http.StatusBadRequest, defines.GENERATE_ACCESSTOKEN_ERROR.Error())
        return
    }

    token := defines.Token{
        AccessToken: accessToken,
        TokenType:   "bearer",
        ExpiresIn:   int(defines.AccessTokenExpireTime / time.Second),
        Scope:       "",
    }

    saveToken(auth.DataManager, client_id, token.AccessToken, token.RefreshToken)

    tokenByte, err := json.Marshal(token)
    if err != nil {
        response.WriteErrorString(http.StatusBadRequest, defines.INTERNAL_ERROR.Error())
        return
    }

    response.Write(tokenByte)
}