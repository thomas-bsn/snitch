from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from typing import cast



def _get_kwargs(
    id: int,
    trial: int,
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/admin/clips/{id}/voice-trials/{trial}".format(id=quote(str(id), safe=""),trial=quote(str(trial), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Any | Error:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Any | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    trial: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Administrateurs uniquement. Supprime un essai de diarisation ; le résultat voix du clip n'est pas
    touché.

    Args:
        id (int):
        trial (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        id=id,
trial=trial,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    trial: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Administrateurs uniquement. Supprime un essai de diarisation ; le résultat voix du clip n'est pas
    touché.

    Args:
        id (int):
        trial (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return sync_detailed(
        id=id,
trial=trial,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    trial: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Any | Error]:
    """  Administrateurs uniquement. Supprime un essai de diarisation ; le résultat voix du clip n'est pas
    touché.

    Args:
        id (int):
        trial (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | Error]
     """


    kwargs = _get_kwargs(
        id=id,
trial=trial,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    trial: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Any | Error | None:
    """  Administrateurs uniquement. Supprime un essai de diarisation ; le résultat voix du clip n'est pas
    touché.

    Args:
        id (int):
        trial (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | Error
     """


    return (await asyncio_detailed(
        id=id,
trial=trial,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
