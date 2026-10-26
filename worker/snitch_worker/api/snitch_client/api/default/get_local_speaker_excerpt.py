from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import File, FileTypes
from ...types import UNSET, Unset
from io import BytesIO
from typing import cast



def _get_kwargs(
    id: int,
    *,
    turn: int | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["turn"] = turn


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/local-speakers/{id}/excerpt".format(id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | File:
    if response.status_code == 200:
        response_200 = File(
             payload = BytesIO(response.content)
        )



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | File]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    turn: int | Unset = UNSET,

) -> Response[Error | File]:
    """  Utilisateurs connectés uniquement, et seulement pour les clips que leur visibilité autorise (sinon
    404). Découpé à la demande depuis le fichier du clip, sans stockage ; Range pris en charge.

    Args:
        id (int):
        turn (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | File]
     """


    kwargs = _get_kwargs(
        id=id,
turn=turn,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    turn: int | Unset = UNSET,

) -> Error | File | None:
    """  Utilisateurs connectés uniquement, et seulement pour les clips que leur visibilité autorise (sinon
    404). Découpé à la demande depuis le fichier du clip, sans stockage ; Range pris en charge.

    Args:
        id (int):
        turn (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | File
     """


    return sync_detailed(
        id=id,
client=client,
turn=turn,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    turn: int | Unset = UNSET,

) -> Response[Error | File]:
    """  Utilisateurs connectés uniquement, et seulement pour les clips que leur visibilité autorise (sinon
    404). Découpé à la demande depuis le fichier du clip, sans stockage ; Range pris en charge.

    Args:
        id (int):
        turn (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | File]
     """


    kwargs = _get_kwargs(
        id=id,
turn=turn,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    turn: int | Unset = UNSET,

) -> Error | File | None:
    """  Utilisateurs connectés uniquement, et seulement pour les clips que leur visibilité autorise (sinon
    404). Découpé à la demande depuis le fichier du clip, sans stockage ; Range pris en charge.

    Args:
        id (int):
        turn (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | File
     """


    return (await asyncio_detailed(
        id=id,
client=client,
turn=turn,

    )).parsed
