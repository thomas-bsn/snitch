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
    code: str,

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/install/{code}".format(code=quote(str(code), safe=""),),
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | str:
    if response.status_code == 200:
        response_200 = response.text
        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | str]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    code: str,
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | str]:
    """  Lanceur d'installation Linux. L'invitation est à usage unique : consommée dès la demande acceptée ;
    404 si inconnue ou expirée, 410 install_invite_used si déjà utilisée. Le script transporte le jeton
    durable du Worker, jamais redonné.

    Args:
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | str]
     """


    kwargs = _get_kwargs(
        code=code,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    code: str,
    *,
    client: AuthenticatedClient | Client,

) -> Error | str | None:
    """  Lanceur d'installation Linux. L'invitation est à usage unique : consommée dès la demande acceptée ;
    404 si inconnue ou expirée, 410 install_invite_used si déjà utilisée. Le script transporte le jeton
    durable du Worker, jamais redonné.

    Args:
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | str
     """


    return sync_detailed(
        code=code,
client=client,

    ).parsed

async def asyncio_detailed(
    code: str,
    *,
    client: AuthenticatedClient | Client,

) -> Response[Error | str]:
    """  Lanceur d'installation Linux. L'invitation est à usage unique : consommée dès la demande acceptée ;
    404 si inconnue ou expirée, 410 install_invite_used si déjà utilisée. Le script transporte le jeton
    durable du Worker, jamais redonné.

    Args:
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | str]
     """


    kwargs = _get_kwargs(
        code=code,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    code: str,
    *,
    client: AuthenticatedClient | Client,

) -> Error | str | None:
    """  Lanceur d'installation Linux. L'invitation est à usage unique : consommée dès la demande acceptée ;
    404 si inconnue ou expirée, 410 install_invite_used si déjà utilisée. Le script transporte le jeton
    durable du Worker, jamais redonné.

    Args:
        code (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | str
     """


    return (await asyncio_detailed(
        code=code,
client=client,

    )).parsed
