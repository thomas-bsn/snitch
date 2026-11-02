from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.list_notifications_filter import ListNotificationsFilter
from ...models.list_notifications_state import ListNotificationsState
from ...models.notification_list import NotificationList
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    state: ListNotificationsState | Unset = ListNotificationsState.ALL,
    filter_: ListNotificationsFilter | Unset = ListNotificationsFilter.ALL,
    limit: int | Unset = 50,
    offset: int | Unset = 0,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_state: str | Unset = UNSET
    if not isinstance(state, Unset):
        json_state = state.value

    params["state"] = json_state

    json_filter_: str | Unset = UNSET
    if not isinstance(filter_, Unset):
        json_filter_ = filter_.value

    params["filter"] = json_filter_

    params["limit"] = limit

    params["offset"] = offset


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/notifications",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | NotificationList:
    if response.status_code == 200:
        response_200 = NotificationList.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | NotificationList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    state: ListNotificationsState | Unset = ListNotificationsState.ALL,
    filter_: ListNotificationsFilter | Unset = ListNotificationsFilter.ALL,
    limit: int | Unset = 50,
    offset: int | Unset = 0,

) -> Response[Error | NotificationList]:
    """  Utilisateurs connectés. Notifications persistantes du demandeur, sur les clips qu'il peut encore
    voir (l'accès est revérifié à chaque lecture, historique compris). unread compte les non-lues,
    pending les actions non résolues : deux compteurs distincts. Aucune écriture.

    Args:
        state (ListNotificationsState | Unset):  Default: ListNotificationsState.ALL.
        filter_ (ListNotificationsFilter | Unset):  Default: ListNotificationsFilter.ALL.
        limit (int | Unset):  Default: 50.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | NotificationList]
     """


    kwargs = _get_kwargs(
        state=state,
filter_=filter_,
limit=limit,
offset=offset,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    state: ListNotificationsState | Unset = ListNotificationsState.ALL,
    filter_: ListNotificationsFilter | Unset = ListNotificationsFilter.ALL,
    limit: int | Unset = 50,
    offset: int | Unset = 0,

) -> Error | NotificationList | None:
    """  Utilisateurs connectés. Notifications persistantes du demandeur, sur les clips qu'il peut encore
    voir (l'accès est revérifié à chaque lecture, historique compris). unread compte les non-lues,
    pending les actions non résolues : deux compteurs distincts. Aucune écriture.

    Args:
        state (ListNotificationsState | Unset):  Default: ListNotificationsState.ALL.
        filter_ (ListNotificationsFilter | Unset):  Default: ListNotificationsFilter.ALL.
        limit (int | Unset):  Default: 50.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | NotificationList
     """


    return sync_detailed(
        client=client,
state=state,
filter_=filter_,
limit=limit,
offset=offset,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    state: ListNotificationsState | Unset = ListNotificationsState.ALL,
    filter_: ListNotificationsFilter | Unset = ListNotificationsFilter.ALL,
    limit: int | Unset = 50,
    offset: int | Unset = 0,

) -> Response[Error | NotificationList]:
    """  Utilisateurs connectés. Notifications persistantes du demandeur, sur les clips qu'il peut encore
    voir (l'accès est revérifié à chaque lecture, historique compris). unread compte les non-lues,
    pending les actions non résolues : deux compteurs distincts. Aucune écriture.

    Args:
        state (ListNotificationsState | Unset):  Default: ListNotificationsState.ALL.
        filter_ (ListNotificationsFilter | Unset):  Default: ListNotificationsFilter.ALL.
        limit (int | Unset):  Default: 50.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | NotificationList]
     """


    kwargs = _get_kwargs(
        state=state,
filter_=filter_,
limit=limit,
offset=offset,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    state: ListNotificationsState | Unset = ListNotificationsState.ALL,
    filter_: ListNotificationsFilter | Unset = ListNotificationsFilter.ALL,
    limit: int | Unset = 50,
    offset: int | Unset = 0,

) -> Error | NotificationList | None:
    """  Utilisateurs connectés. Notifications persistantes du demandeur, sur les clips qu'il peut encore
    voir (l'accès est revérifié à chaque lecture, historique compris). unread compte les non-lues,
    pending les actions non résolues : deux compteurs distincts. Aucune écriture.

    Args:
        state (ListNotificationsState | Unset):  Default: ListNotificationsState.ALL.
        filter_ (ListNotificationsFilter | Unset):  Default: ListNotificationsFilter.ALL.
        limit (int | Unset):  Default: 50.
        offset (int | Unset):  Default: 0.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | NotificationList
     """


    return (await asyncio_detailed(
        client=client,
state=state,
filter_=filter_,
limit=limit,
offset=offset,

    )).parsed
